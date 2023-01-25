package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClusterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccExampleResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kind_cluster.test", "id", "test-cluster/v1.26.0"),
					resource.TestCheckResourceAttr("kind_cluster.test", "name", "test-cluster"),
					resource.TestCheckResourceAttr("kind_cluster.test", "node_version", "v1.26.0"),
					resource.TestCheckResourceAttrSet("kind_cluster.test", "kubeconfig"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "kind_cluster.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			// {
			// 	Config: testAccExampleResourceConfig("two"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttr("scaffolding_example.test", "configurable_attribute", "two"),
			// 	),
			// },
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccExampleResourceConfig(configurableAttribute string) string {
	return `
resource "kind_cluster" "test" {
  name = "test-cluster"
  node_version = "v1.26.0"
}`
}
