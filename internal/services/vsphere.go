package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	api "github.com/kubev2v/forklift/pkg/apis/forklift/v1beta1"
	"github.com/kubev2v/forklift/pkg/controller/provider/container/vsphere"
	"github.com/kubev2v/forklift/pkg/controller/provider/model"
	webprovider "github.com/kubev2v/forklift/pkg/controller/provider/web"
	"github.com/kubev2v/forklift/pkg/controller/provider/web/base"
	web "github.com/kubev2v/forklift/pkg/controller/provider/web/vsphere"
	libcontainer "github.com/kubev2v/forklift/pkg/lib/inventory/container"
	libmodel "github.com/kubev2v/forklift/pkg/lib/inventory/model"
	libweb "github.com/kubev2v/forklift/pkg/lib/inventory/web"
	"go.uber.org/zap"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

// VSphereCollector wraps the forklift vSphere collector.
type VSphereCollector struct {
	collector *vsphere.Collector
	container *libcontainer.Container
	db        libmodel.DB
	dbPath    string
}

func NewVSphereCollector(creds *models.Credentials, dataDir string) (*VSphereCollector, error) {
	provider := createProvider(creds)
	secret := createSecret(creds)

	dbPath := filepath.Join(dataDir, "vsphere.db")
	db, err := createDB(provider, dbPath)
	if err != nil {
		return nil, err
	}

	collector := vsphere.New(db, provider, secret)

	return &VSphereCollector{
		collector: collector,
		db:        db,
		dbPath:    dbPath,
	}, nil
}

// Collect runs the vSphere collection process.
// This starts the forklift collector which populates the SQLite database.
// The method blocks until collection is complete or the context is cancelled.
func (c *VSphereCollector) Collect(ctx context.Context) error {
	zap.S().Info("starting forklift vSphere collector")

	// Start the web container and wait for collection to complete
	container, err := startWebContainer(c.collector)
	if err != nil {
		return err
	}
	c.container = container

	zap.S().Info("forklift vSphere collection completed (parity reached)")
	return nil
}

// DBPath returns the path to the SQLite database.
func (c *VSphereCollector) DBPath() string {
	return c.dbPath
}

// ForkliftCollector returns the underlying forklift vSphere collector.
// This is needed by the inventory builder to access the collected data.
func (c *VSphereCollector) ForkliftCollector() *vsphere.Collector {
	return c.collector
}

// Close cleans up collector resources.
func (c *VSphereCollector) Close() {
	if c.container != nil {
		c.container.Delete(c.collector.Owner())
	}
	if c.db != nil {
		_ = c.db.Close(true)
	}
}

// createProvider creates a forklift Provider object from credentials.
func createProvider(creds *models.Credentials) *api.Provider {
	vsphereType := api.VSphere
	return &api.Provider{
		ObjectMeta: meta.ObjectMeta{
			UID: "1",
		},
		Spec: api.ProviderSpec{
			URL:  creds.URL,
			Type: &vsphereType,
		},
	}
}

// createSecret creates a Kubernetes Secret with vCenter credentials.
func createSecret(creds *models.Credentials) *core.Secret {
	return &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      "vsphere-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"user":               []byte(creds.Username),
			"password":           []byte(creds.Password),
			"insecureSkipVerify": []byte("true"),
		},
	}
}

// createDB creates the SQLite database for the collector.
func createDB(provider *api.Provider, path string) (libmodel.DB, error) {
	models := model.Models(provider)
	db := libmodel.New(path, models...)
	if err := db.Open(true); err != nil {
		return nil, err
	}
	return db, nil
}

// startWebContainer starts the forklift web container which triggers collection.
// It blocks until the collector reaches parity (fully synchronized with vCenter).
func startWebContainer(collector *vsphere.Collector) (*libcontainer.Container, error) {
	container := libcontainer.New()
	if err := container.Add(collector); err != nil {
		return nil, err
	}

	handlers := []libweb.RequestHandler{
		&libweb.SchemaHandler{},
		&webprovider.ProviderHandler{
			Handler: base.Handler{
				Container: container,
			},
		},
	}
	handlers = append(handlers, web.Handlers(container)...)

	webServer := libweb.New(container, handlers...)
	webServer.Start()

	// Wait for collector to reach parity (fully synchronized with vCenter)
	// This matches the migration-planner implementation
	const maxRetries = 300 // 5 minutes timeout (300 * 1 second)
	for i := 0; i < maxRetries; i++ {
		time.Sleep(1 * time.Second)
		if collector.HasParity() {
			zap.S().Debug("collector reached parity")
			return container, nil
		}
		if i > 0 && i%30 == 0 {
			zap.S().Infof("waiting for vSphere collection... (%d seconds)", i)
		}
	}

	return container, fmt.Errorf("timed out waiting for collector parity")
}
