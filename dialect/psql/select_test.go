package psql

import (
	"testing"

	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/query"
)

func TestSelect(t *testing.T) {
	var qm = SelectQM{}

	tests := map[string]struct {
		query         query.Query
		expectedQuery string
		expectedArgs  []any
	}{
		"simple select": {
			expectedQuery: "SELECT id, name FROM users WHERE id IN ($1, $2, $3)",
			expectedArgs:  []any{100, 200, 300},
			query: Select(
				qm.Select("id", "name"),
				qm.From("users"),
				qm.Where(expr.IN("id", expr.Arg(100, 200, 300))),
			),
		},
		"with rows from": {
			query: Select(
				qm.From(
					expr.Func(
						"json_to_recordset",
						expr.Arg(`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`),
					).Col("a", "INTEGER").Col("b", "TEXT").ToMod(),
					expr.Func("generate_series", 1, 3).ToMod(),
					qm.As("x", "p", "q", "s"),
				),
				qm.OrderBy("p"),
			),
			expectedQuery: ` SELECT *
				FROM ROWS FROM
					(
						json_to_recordset($1)
							AS (a INTEGER, b TEXT),
						generate_series(1, 3)
					) AS "x" ("p", "q", "s")
				ORDER BY p`,
			expectedArgs: []any{`[{"a":40,"b":"foo"},{"a":"100","b":"bar"}]`},
		},
		"with sub-select": {
			expectedQuery: `SELECT status, avg(difference)
					FROM (
						SELECT
						status,
						LEAD(created_date, 1, NOW())
						OVER(PARTITION BY presale_id ORDER BY created_date) -
						created_date AS "difference"
						FROM presales_presalestatus
					) AS "differnce_by_status"
					WHERE status IN ('A', 'B', 'C')
					GROUP BY status`,
			query: Select(
				qm.Select("status", expr.Func("avg", "difference")),
				qm.From(
					Select(
						qm.Select(
							"status",
							expr.C(expr.MINUS(expr.OVER(
								expr.Func("LEAD", "created_date", 1, expr.Func("NOW")),
								expr.Window("").PartitionBy("presale_id").OrderBy("created_date"),
							), "created_date"), "difference"),
						),
						qm.From("presales_presalestatus"),
					),
					qm.As("differnce_by_status"),
				),
				qm.Where(expr.IN("status", expr.S("A"), expr.S("B"), expr.S("C"))),
				qm.GroupBy("status"),
			),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sql, args, err := query.Build(tc.query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff := queryDiff(tc.expectedQuery, sql); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff := argsDiff(tc.expectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
