package vmware

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/vim25"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// ValidateUserPrivilegesOnEntity checks whether the specified user has all the required privileges
// on a given vSphere entity (e.g., VM, folder, datacenter).
func ValidateUserPrivilegesOnEntity(
	ctx context.Context,
	client *vim25.Client,
	ref types.ManagedObjectReference,
	requiredPrivileges []string,
	username string,
) error {
	authManager := object.NewAuthorizationManager(client)

	results, err := authManager.FetchUserPrivilegeOnEntities(ctx, []types.ManagedObjectReference{ref}, username)
	if err != nil {
		return fmt.Errorf("failed to fetch user privileges: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no privileges returned for user %s", username)
	}

	grantedMap := make(map[string]bool)
	for _, p := range results[0].Privileges {
		grantedMap[p] = true
	}

	var missing []string
	for _, req := range requiredPrivileges {
		if !grantedMap[req] {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("user %s is missing required privileges: %v", username, missing)
	}

	return nil
}

func (m *VMManager) ValidatePrivileges(ctx context.Context, moid string, requiredPrivileges []string) error {
	return ValidateUserPrivilegesOnEntity(ctx, m.gc.Client, refFromMoid(moid), requiredPrivileges, m.username)
}
