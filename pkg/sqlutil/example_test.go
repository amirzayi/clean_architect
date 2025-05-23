package sqlutil_test

import (
	"fmt"

	"github.com/amirzayi/clean_architect/pkg/paginate"
	"github.com/amirzayi/clean_architect/pkg/sqlutil"
)

func ExampleBuildPaginationQuery() {
	query, args := sqlutil.BuildPaginationQuery("user", &paginate.Pagination{
		Page:    3,
		PerPage: 15,
		Fields:  []string{"name", "id", "phone", "role", "status"},
		Sort: []paginate.Sort{
			{Field: "id", Arrange: paginate.SortOrderDescending},
			{Field: "name", Arrange: paginate.SortOrderAscending},
		},
		Filters: []paginate.Filter{
			{Key: "name", Value: "amir,admin,test", Condition: paginate.FilterIn},
			{Key: "status", Value: "1,2", Condition: paginate.FilterIn},
		},
	}, map[string]string{
		"id":         "id",
		"name":       "name",
		"phone":      "phone",
		"email":      "email",
		"status":     "status",
		"role":       "role",
		"created_at": "created_at",
	})
	fmt.Println(query)
	fmt.Println(args)

	// Output:
	// SELECT name,id,phone,role,status FROM user
	// WHERE name IN(?,?,?) AND status IN(?,?)
	// ORDER BY id desc, name asc
	// LIMIT ? offset ?
	// [amir admin test 1 2 15 30]
}
