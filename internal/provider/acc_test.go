package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAcc runs against a real Kayra, KAYRA_URL with KAYRA_TOKEN, a
// sysadmin's: TF_ACC=1 make testacc, OpenTofu or Terraform on the PATH
// (TF_ACC_TERRAFORM_PATH, TF_ACC_PROVIDER_NAMESPACE=infrapi). It creates
// a namespace and its RBAC, updates one attribute, imports them, and
// deletes them.
func TestAcc(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC unset: acceptance tests run against a real Kayra")
	}
	for _, env := range []string{"KAYRA_URL", "KAYRA_TOKEN"} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s is unset", env)
		}
	}
	ns := fmt.Sprintf("acc-%d", time.Now().Unix())
	config := func(description string) string {
		return fmt.Sprintf(`
resource "kayra_namespace" "acc" {
  name = %[1]q
}

resource "kayra_group" "admins" {
  namespace   = kayra_namespace.acc.name
  name        = "admins"
  description = %[2]q
  users       = ["jdoe@example.com"]
}

resource "kayra_role" "admin" {
  namespace = kayra_namespace.acc.name
  name      = "admin"
  rules = [{
    apis      = ["*"]
    resources = ["*"]
    verbs     = ["get", "list"]
  }]
}

resource "kayra_role_binding" "admins" {
  namespace = kayra_namespace.acc.name
  name      = "admins"
  role      = kayra_role.admin.name
  groups    = [kayra_group.admins.name]
}
`, ns, description)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"kayra": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{Config: config("the admins"), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("kayra_group.admins", "id", ns+"/admins"),
				resource.TestCheckResourceAttr("kayra_group.admins", "metadata.namespace", ns),
				resource.TestCheckResourceAttrSet("kayra_namespace.acc", "created_at"),
			)},
			{Config: config("the namespace's admins"), Check: resource.TestCheckResourceAttr("kayra_group.admins", "description", "the namespace's admins")},
			{ResourceName: "kayra_group.admins", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"updated_at"}},
			{ResourceName: "kayra_role_binding.admins", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"updated_at"}},
		},
	})
}
