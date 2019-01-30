package resources_test

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func MustParseInt(input string) int64 {
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

func extractList(in map[string]string, name string) ([]string, error) {
	out := make([]string, 0)
	r, err := regexp.Compile(fmt.Sprintf(`^%s.\d+(.+)$`, name))
	if err != nil {
		return out, err
	}
	for k, v := range in {
		if r.MatchString(k) {
			log.Printf("[DEBUG] matched %s %s", k, v)
			out = append(out, v)
		} else {
			log.Printf("[DEBUG] no match %s", k)
		}
	}
	return out, nil
}

func listEquiv(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if strings.ToUpper(a[i]) != strings.ToUpper(b[i]) {
			return false
		}
	}
	return true
}

func testCheckRolesAndUsers(path string, roles, users []string) func(state *terraform.State) error {

	return func(state *terraform.State) error {
		is := state.RootModule().Resources[path].Primary

		if c, ok := is.Attributes["roles.#"]; !ok || MustParseInt(c) != int64(len(roles)) {
			return fmt.Errorf("expected roles.# to equal %d but got %s", len(roles), c)
		}
		r, err := extractList(is.Attributes, "roles")
		if err != nil {
			return err
		}

		if !listEquiv(roles, r) {
			return fmt.Errorf("expected roles %#v but got %#v", roles, r)
		}

		if c, ok := is.Attributes["users.#"]; !ok || MustParseInt(c) != int64(len(users)) {
			return fmt.Errorf("expected users.# to equal %d but got %s", len(users), c)
		}
		u, err := extractList(is.Attributes, "users")
		if err != nil {
			return err
		}

		if !listEquiv(users, u) {
			return fmt.Errorf("expected users %#v but got %#v", users, u)
		}

		return nil
	}
}

func TestAccGrantRole(t *testing.T) {
	role1 := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	role2 := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	user1 := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	basicChecks := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("snowflake_role.r", "name", strings.ToUpper(role1)),
		resource.TestCheckResourceAttr("snowflake_role.r2", "name", strings.ToUpper(role2)),
		resource.TestCheckResourceAttr("snowflake_role_grants.w", "name", strings.ToUpper(role1)),
	)

	baselineStep := resource.TestStep{
		Config:       rgConfig(role1, role2, user1),
		ResourceName: "snowflake_role_grants.w",
		Check: resource.ComposeTestCheckFunc(
			basicChecks,
			testCheckRolesAndUsers("snowflake_role_grants.w", []string{role2}, []string{user1}),
		),
	}

	resource.Test(t, resource.TestCase{
		Providers: providers(),
		Steps: []resource.TestStep{
			// test settup + removing a role
			baselineStep,
			{
				Config:       rgConfig2(role1, role2, user1),
				ResourceName: "snowflake_role_grants.w",
				Check: resource.ComposeTestCheckFunc(
					basicChecks,
					testCheckRolesAndUsers("snowflake_role_grants.w", []string{}, []string{user1}),
				),
			},
			// back to baseline, which means adding a role, then remove a user
			baselineStep,
			{
				Config:       rgConfig3(role1, role2, user1),
				ResourceName: "snowflake_role_grants.w",

				Check: resource.ComposeTestCheckFunc(
					basicChecks,
					testCheckRolesAndUsers("snowflake_role_grants.w", []string{role2}, []string{}),
				),
			},
			// add the role back to get back to baseline
			baselineStep,
			// 			// IMPORT
			// 			{
			// 				ResourceName:            "snowflake_role_grants.w",
			// 				ImportState:             true,
			// 				ImportStateVerify:       true,
			// 			},
		},
	})
}

func rgConfig(role1, role2, user1 string) string {
	s := `
resource "snowflake_role" "r" {
	name = "%s"
}
resource "snowflake_role" "r2" {
	name = "%s"
}
resource "snowflake_user" "u" {
	name = "%s"
}
resource "snowflake_role_grants" "w" {
	name = "${snowflake_role.r.name}"
	role_name = "${snowflake_role.r.name}"
	roles = ["${snowflake_role.r2.name}"]
	users = ["${snowflake_user.u.name}"]
}
`
	return fmt.Sprintf(s, role1, role2, user1)
}

func rgConfig2(role1, role2, user1 string) string {
	s := `
resource "snowflake_role" "r" {
	name = "%s"
}
resource "snowflake_role" "r2" {
	name = "%s"
}
resource "snowflake_user" "u" {
	name = "%s"
}
resource "snowflake_role_grants" "w" {
	name = "${snowflake_role.r.name}"
	role_name = "${snowflake_role.r.name}"
	users = ["${snowflake_user.u.name}"]
}
`
	return fmt.Sprintf(s, role1, role2, user1)
}

func rgConfig3(role1, role2, user1 string) string {
	s := `
resource "snowflake_role" "r" {
	name = "%s"
}
resource "snowflake_role" "r2" {
	name = "%s"
}
resource "snowflake_user" "u" {
	name = "%s"
}
resource "snowflake_role_grants" "w" {
	name = "${snowflake_role.r.name}"
	role_name = "${snowflake_role.r.name}"
	roles = ["${snowflake_role.r2.name}"]
}
`
	return fmt.Sprintf(s, role1, role2, user1)
}

func rgConfig4(role1, role2, user1 string) string {
	s := `
resource "snowflake_role" "r" {
	name = "%s"
}
resource "snowflake_role" "r2" {
	name = "%s"
}
resource "snowflake_user" "u" {
	name = "%s"
}
resource "snowflake_role_grants" "w" {
	name = "${snowflake_role.r.name}"
	role_name = "${snowflake_role.r.name}"
	roles = ["${snowflake_role.r2.name}"]
	users = ["${snowflake_user.u.name}"]
}
`
	return fmt.Sprintf(s, role1, role2, user1)
}
