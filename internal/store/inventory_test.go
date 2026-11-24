package store_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubev2v/migration-planner/pkg/duckdb_parser"

	"github.com/kubev2v/assisted-migration-agent/internal/store"
	"github.com/kubev2v/assisted-migration-agent/internal/store/migrations"
	srvErrors "github.com/kubev2v/assisted-migration-agent/pkg/errors"
	"github.com/kubev2v/assisted-migration-agent/test"
)

var _ = Describe("InventoryStore", func() {
	var (
		ctx    context.Context
		s      *store.Store
		pool   *store.Pool
		tmpDir string
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		tmpDir, err = os.MkdirTemp("", "inventory-store-test-*")
		Expect(err).NotTo(HaveOccurred())
		pool = store.NewPool(5 * time.Minute)
		collDB, dbErr := pool.NewDatabase("coll", filepath.Join(tmpDir, "collection.duckdb"), time.Now(), store.EagerConnectionInitilization, 0, store.ReadWriteDatabase)
		Expect(dbErr).NotTo(HaveOccurred())
		Expect(collDB.Migrate(ctx, func(mCtx context.Context, sqlDB *sql.DB) error {
			st, stErr := collDB.Store()
			if stErr != nil {
				return stErr
			}
			if pErr := duckdb_parser.New(st.Querier(), test.NewMockValidator()).Init(); pErr != nil {
				return pErr
			}
			return migrations.RunCollection(mCtx, sqlDB, "collection")
		})).To(Succeed())
		pool.Add(collDB)
		s, err = collDB.Store()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if pool != nil {
			pool.Close()
		}
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("Delete", func() {
		BeforeEach(func() {
			Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-one"}`))).To(Succeed())
			Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-two"}`))).To(Succeed())
		})

		It("deletes only the matching inventory", func() {
			Expect(s.Inventory().Delete(ctx, sq.Eq{"id": "vc-one"})).To(Succeed())

			var ids []string
			rows, err := s.Querier().QueryContext(ctx, `SELECT id FROM inventory`)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(rows.Close)
			for rows.Next() {
				var id string
				Expect(rows.Scan(&id)).To(Succeed())
				ids = append(ids, id)
			}
			Expect(rows.Err()).NotTo(HaveOccurred())
			Expect(ids).To(ConsistOf("vc-two"))
		})

		It("deletes all inventories when no filter is provided", func() {
			Expect(s.Inventory().Delete(ctx, nil)).To(Succeed())

			var count int
			err := s.Querier().QueryRowContext(ctx, `SELECT count(*) FROM inventory`).Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeZero())
		})
	})

	Context("Infrastructure sources", func() {
		It("preserves other sources when recording and deleting infrastructure", func() {
			for _, table := range []string{"vhost", "vdatastore", "vhba", "vhostscsi", "vnic"} {
				_, err := s.Querier().ExecContext(ctx, "INSERT INTO "+table+" (source_vcenter_id) VALUES ('vc-existing'), (NULL)")
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(s.Inventory().SetInfrastructureSource(ctx, "vc-new")).To(Succeed())
			Expect(s.Inventory().DeleteInfrastructure(ctx, "vc-new")).To(Succeed())
			for _, table := range []string{"vhost", "vdatastore", "vhba", "vhostscsi", "vnic"} {
				var owner string
				Expect(s.Querier().QueryRowContext(ctx, "SELECT source_vcenter_id FROM "+table).Scan(&owner)).To(Succeed())
				Expect(owner).To(Equal("vc-existing"))
				var count int
				Expect(s.Querier().QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count)).To(Succeed())
				Expect(count).To(Equal(1))
			}
		})

		DescribeTable("preserves ambiguous legacy datastore ownership",
			func(secondName string) {
				Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-one","vcenter":{"infra":{"hosts":[{"id":"host-1"}],"datastores":[{"diskId":"shared"}]}}}`))).To(Succeed())
				Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-two","vcenter":{"infra":{"hosts":[],"datastores":[{"diskId":"`+secondName+`"}]}}}`))).To(Succeed())
				_, err := s.Querier().ExecContext(ctx, `
					INSERT INTO vhost ("Host", "Object ID", "VI SDK UUID") VALUES ('esxi-shared', 'host-1', 'vc-one');
					INSERT INTO vdatastore ("Name", "Hosts") VALUES ('shared', 'esxi-shared'), ('shared', NULL);
				`)
				Expect(err).NotTo(HaveOccurred())
				Expect(s.Inventory().DeleteInfrastructure(ctx, "vc-one")).To(Succeed())
				var count int
				Expect(s.Querier().QueryRowContext(ctx, "SELECT count(*) FROM vdatastore").Scan(&count)).To(Succeed())
				Expect(count).To(Equal(2), "an absent host sheet cannot establish datastore ownership")

				Expect(s.Inventory().Delete(ctx, sq.Eq{"id": "vc-one"})).To(Succeed())
				Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-third"}`))).To(Succeed())
				Expect(s.Inventory().DeleteInfrastructure(ctx, "vc-two")).To(Succeed())
				Expect(s.Querier().QueryRowContext(ctx, "SELECT count(*) FROM vdatastore").Scan(&count)).To(Succeed())
				Expect(count).To(Equal(2), "removing a source must not turn missing evidence into ownership")
			},
			Entry("when another source has the same name but no host sheet", "shared"),
			Entry("when raw rows outnumber the saved source snapshot", "other"),
		)

		It("preserves legacy host metadata without recorded source ownership", func() {
			Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-one"}`))).To(Succeed())
			Expect(s.Inventory().SaveSubset(ctx, []byte(`{"vcenter_id":"vc-two"}`))).To(Succeed())
			_, err := s.Querier().ExecContext(ctx, `INSERT INTO vhost ("Object ID", "VI SDK UUID") VALUES ('host-1', 'vc-one')`)
			Expect(err).NotTo(HaveOccurred())
			for _, table := range []string{"vhba", "vhostscsi", "vnic"} {
				_, err := s.Querier().ExecContext(ctx, "INSERT INTO "+table+` ("Host ID") VALUES ('host-1'), ('host-1')`)
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(s.Inventory().DeleteInfrastructure(ctx, "vc-one")).To(Succeed())
			for _, table := range []string{"vhba", "vhostscsi", "vnic"} {
				var count int
				Expect(s.Querier().QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count)).To(Succeed())
				Expect(count).To(Equal(2))
			}
		})

		It("deletes infrastructure when an old collection has no NIC table", func() {
			_, err := s.Querier().ExecContext(ctx, `
				DROP TABLE vnic;
				INSERT INTO vhost (source_vcenter_id) VALUES ('vc-one');
			`)
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Inventory().DeleteInfrastructure(ctx, "vc-one")).To(Succeed())
			var count int
			Expect(s.Querier().QueryRowContext(ctx, "SELECT count(*) FROM vhost").Scan(&count)).To(Succeed())
			Expect(count).To(BeZero())
		})

		It("rolls infrastructure deletion back with its transaction", func() {
			for _, table := range []string{"vhost", "vdatastore", "vhba", "vhostscsi", "vnic"} {
				_, err := s.Querier().ExecContext(ctx, "INSERT INTO "+table+" (source_vcenter_id) VALUES ('vc-one')")
				Expect(err).NotTo(HaveOccurred())
			}
			err := s.WithTx(ctx, func(txCtx context.Context) error {
				if err := s.Inventory().DeleteInfrastructure(txCtx, "vc-one"); err != nil {
					return err
				}
				return context.Canceled
			})
			Expect(err).To(MatchError(context.Canceled))
			for _, table := range []string{"vhost", "vdatastore", "vhba", "vhostscsi", "vnic"} {
				var count int
				Expect(s.Querier().QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count)).To(Succeed())
				Expect(count).To(Equal(1))
			}
		})
	})

	Context("Save", func() {
		// Given inventory data with a vCenter ID
		// When we save the main inventory
		// Then its database ID should be the vCenter ID
		It("should store the vCenter ID as the main inventory ID", func() {
			// Arrange
			data := []byte(`{
				"vcenter_id": "vc-main",
				"vcenter_version": "8.0.3",
				"vcenter": {
					"vms": {"total": 2},
					"infra": {"totalHosts": 1}
				},
				"clusters": {}
			}`)

			// Act
			err := s.Inventory().Save(ctx, data)
			Expect(err).NotTo(HaveOccurred())

			// Assert
			var id string
			err = s.Querier().QueryRowContext(ctx, `SELECT id FROM inventory WHERE is_subset = false`).Scan(&id)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("vc-main"))
		})

		// Given valid inventory data
		// When we save the inventory
		// Then it should save successfully without error
		It("should save inventory successfully", func() {
			// Arrange
			data := []byte(`{"vms": [{"name": "vm1"}]}`)

			// Act
			err := s.Inventory().Save(ctx, data)

			// Assert
			Expect(err).NotTo(HaveOccurred())
		})

		// Given existing inventory in the store
		// When we save new inventory data
		// Then it should update the existing record (upsert)
		It("should update inventory on second save (upsert)", func() {
			// Arrange
			data1 := []byte(`{"vms": [{"name": "vm1"}]}`)
			err := s.Inventory().Save(ctx, data1)
			Expect(err).NotTo(HaveOccurred())

			// Act
			data2 := []byte(`{"vms": [{"name": "vm1"}, {"name": "vm2"}]}`)
			err = s.Inventory().Save(ctx, data2)
			Expect(err).NotTo(HaveOccurred())

			// Assert
			retrieved, err := s.Inventory().Get(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Data).To(Equal(data2))
		})
	})

	Context("SaveSubset", func() {
		// Given inventory data with a vCenter ID
		// When we save a subset inventory
		// Then its database ID should be the vCenter ID
		It("should store the vCenter ID as the subset inventory ID", func() {
			// Arrange
			data := []byte(`{
				"vcenter_id": "vc-subset",
				"vcenter_version": "8.0.2",
				"vcenter": {
					"vms": {"total": 3},
					"infra": {"totalHosts": 2}
				},
				"clusters": {}
			}`)

			// Act
			err := s.Inventory().SaveSubset(ctx, data)
			Expect(err).NotTo(HaveOccurred())

			// Assert
			var id string
			err = s.Querier().QueryRowContext(ctx, `SELECT id FROM inventory WHERE is_subset = true`).Scan(&id)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("vc-subset"))
		})
	})

	Context("Get", func() {
		// Given an empty inventory store
		// When we try to get the inventory
		// Then it should return ResourceNotFoundError
		It("should return ResourceNotFoundError when no inventory exists", func() {
			// Act
			_, err := s.Inventory().Get(ctx)

			// Assert
			Expect(srvErrors.IsResourceNotFoundError(err)).To(BeTrue())
		})

		// Given saved inventory in the store
		// When we retrieve the inventory
		// Then it should return the saved data
		It("should retrieve saved inventory", func() {
			// Arrange
			data := []byte(`{"vms": [{"name": "vm1"}]}`)
			err := s.Inventory().Save(ctx, data)
			Expect(err).NotTo(HaveOccurred())

			// Act
			retrieved, err := s.Inventory().Get(ctx)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Data).To(Equal(data))
		})

		// Given saved inventory in the store
		// When we retrieve the inventory
		// Then it should have timestamps set by the database
		It("should have timestamps set by database", func() {
			// Arrange
			data := []byte(`{"vms": []}`)
			err := s.Inventory().Save(ctx, data)
			Expect(err).NotTo(HaveOccurred())

			// Act
			retrieved, err := s.Inventory().Get(ctx)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.CreatedAt).NotTo(BeZero())
			Expect(retrieved.UpdatedAt).NotTo(BeZero())
		})
	})
})
