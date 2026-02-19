// Package filter provides a DSL for filtering VMs using SQL-like expressions.
//
// The filter package implements a lexer, parser, and SQL generator that converts
// human-readable filter expressions into squirrel Sqlizer objects for use with
// SelectBuilder queries.
//
// # Grammar
//
//	expression  : term ( "or" term )* ;
//	term        : factor ( "and" factor )* ;
//	factor      : equality | "(" expression ")" ;
//	equality    : IDENTIFIER ( "=" | "!=" | "<" | "<=" | ">" | ">=" ) value
//	            | IDENTIFIER ( "~" | "!~" ) REGEX_LITERAL
//	            | IDENTIFIER "in" "[" STRING ( "," STRING )* "]"
//	            | IDENTIFIER "not" "in" "[" STRING ( "," STRING )* "]" ;
//	value       : STRING | QUANTITY | BOOLEAN ;
//
//	IDENTIFIER    : [a-zA-Z_][a-zA-Z0-9_.]* ;
//	REGEX_LITERAL : '/' ( '\\/' | . )*? '/' ;
//	STRING        : "'" (.*?) "'" | '"' (.*?) '"' ;
//	BOOLEAN       : "true" | "false" ;
//	QUANTITY      : [0-9]+(\.[0-9]+)? ( 'KB' | 'MB' | 'GB' | 'TB' )? ;
//
// # Operators
//
//	=    Equal
//	!=   Not equal
//	>    Greater than
//	>=   Greater than or equal
//	<    Less than
//	<=   Less than or equal
//	~      Regex match (uses regexp_matches)
//	!~     Regex not match
//	in     Membership test (SQL IN clause)
//	not in Exclusion test (SQL NOT IN clause)
//	and    Logical AND (higher precedence than OR)
//	or     Logical OR
//
// # Value Types
//
// Strings: Single or double quoted. Empty strings are allowed.
//
//	name = 'production'
//	name = "test-vm"
//	description = ''
//
// Booleans: Case-insensitive true/false.
//
//	active = true
//	enabled = FALSE
//
// Quantities: Numbers with optional size units (KB, MB, GB, TB).
// All quantities are normalized to MB for comparison.
//
//	memory > 8GB        // 8192 MB
//	disk >= 1TB         // 1048576 MB
//	memory < 512MB      // 512 MB
//	memory > 1024KB     // 1 MB
//	count = 100         // plain number (no conversion)
//
// Regex: AWK-style patterns between forward slashes.
//
//	name ~ /^prod-.*/           // starts with "prod-"
//	name ~ /web|api/            // contains "web" or "api"
//	name !~ /test/              // does not contain "test"
//	path ~ /a\/b/               // escaped slash matches "a/b"
//
// Lists: Comma-separated strings in square brackets for IN/NOT IN operators.
//
//	status in ['active', 'pending', 'running']
//	cluster in ['prod', 'staging']
//	status not in ['deleted', 'archived']
//	name not in ['test-vm', 'dev-vm']
//
// # Identifiers
//
// Identifiers support dotted notation for nested fields:
//
//	vm.name = 'test'
//	vm.host.datacenter = 'DC1'
//	config.nested.value > 100
//
// # Operator Precedence
//
// AND binds tighter than OR. Use parentheses to override:
//
//	a = '1' or b = '2' and c = '3'       // a OR (b AND c)
//	(a = '1' or b = '2') and c = '3'     // (a OR b) AND c
//
// # Usage with squirrel SelectBuilder
//
// The Parse function returns a squirrel.Sqlizer that can be used with
// SelectBuilder.Where():
//
//	import (
//	    sq "github.com/Masterminds/squirrel"
//	    "github.com/kubev2v/assisted-migration-agent/pkg/filter"
//	)
//
//	// Define a mapper from filter field names to SQL columns
//	mapper := filter.MapFunc(func(name string) (string, error) {
//	    switch name {
//	    case "name":
//	        return `v."VM"`, nil
//	    case "memory":
//	        return `v."Memory"`, nil
//	    case "cluster":
//	        return `v."Cluster"`, nil
//	    case "status":
//	        return `v."Powerstate"`, nil
//	    default:
//	        return "", fmt.Errorf("unknown field: %s", name)
//	    }
//	})
//
//	// Parse the filter expression
//	sqlizer, err := filter.Parse([]byte("memory > 8GB and status = 'poweredOn'"), mapper)
//	if err != nil {
//	    return err
//	}
//
//	// Use with SelectBuilder
//	query, args, err := sq.Select("*").
//	    From("vms").
//	    Where(sqlizer).
//	    ToSql()
//	// query: SELECT * FROM vms WHERE ((v."Memory" > ?) AND (v."Powerstate" = ?))
//	// args: [8192.00, "poweredOn"]
//
// IN operator generates SQL IN clauses:
//
//	sqlizer, _ := filter.Parse([]byte("status in ['poweredOn', 'suspended']"), mapper)
//	query, args, _ := sq.Select("*").From("vms").Where(sqlizer).ToSql()
//	// query: SELECT * FROM vms WHERE v."Powerstate" IN (?,?)
//	// args: ["poweredOn", "suspended"]
//
// # Creating a ListOption
//
// Common pattern for integrating with store queries:
//
//	func ByFilter(filterStr string, mapper filter.MapFunc) store.ListOption {
//	    return func(b sq.SelectBuilder) sq.SelectBuilder {
//	        if filterStr == "" {
//	            return b
//	        }
//	        sqlizer, err := filter.Parse([]byte(filterStr), mapper)
//	        if err != nil {
//	            // Log error and return unmodified builder
//	            return b
//	        }
//	        return b.Where(sqlizer)
//	    }
//	}
//
//	// Usage:
//	vms, err := store.VM().List(ctx,
//	    ByFilter("memory > 8GB and cluster = 'prod'", vmMapper),
//	    store.WithLimit(50),
//	)
//
// # Filter Examples
//
// Simple comparisons:
//
//	name = 'web-server-01'
//	status != 'poweredOff'
//	memory >= 16GB
//	disk < 500GB
//
// Regex matching:
//
//	name ~ /^prod-/              // starts with "prod-"
//	name ~ /-(dev|test)-/        // contains "-dev-" or "-test-"
//	name !~ /backup/             // excludes "backup"
//
// Boolean filters:
//
//	active = true
//	template = false
//
// IN/NOT IN filters (membership test):
//
//	status in ['poweredOn', 'suspended']
//	cluster in ['prod', 'staging', 'dev']
//	status not in ['deleted', 'archived']
//	name not in ['test-vm', 'backup-vm']
//
// Combined filters:
//
//	memory >= 8GB and disk >= 100GB
//	status = 'poweredOn' or status = 'suspended'
//	name ~ /^prod-/ and memory >= 16GB and active = true
//
// Complex expressions with grouping:
//
//	(cluster = 'prod' or cluster = 'staging') and memory >= 8GB
//	active = true and (memory >= 16GB or disk >= 500GB)
//	(name ~ /^web-/ or name ~ /^api-/) and status = 'poweredOn' and memory >= 4GB
//
// IN with other conditions:
//
//	status in ['poweredOn', 'suspended'] and memory >= 8GB
//	cluster in ['prod', 'staging'] and name ~ /^web-/
//
// # Error Handling
//
// Parse returns a ParseError with position information on syntax errors:
//
//	_, err := filter.Parse([]byte("name ="), mapper)
//	// err: parse error at 6: expected value instead of eol
//
//	_, err := filter.Parse([]byte("name ~ /unclosed"), mapper)
//	// err: parse error at 7: unclosed regex
//
// The MapFunc can return errors for unknown fields:
//
//	mapper := func(name string) (string, error) {
//	    if name == "unknown" {
//	        return "", fmt.Errorf("unknown field: %s", name)
//	    }
//	    return `"` + name + `"`, nil
//	}
//
// // {{- /*
// -// Flattened VM Query — joins all tables without aggregation so every column
// -// is directly available in the WHERE clause for filtering.
// -//
// -// A VM with N disks and M NICs produces N×M rows (cartesian product).
// -// Use SELECT DISTINCT i."VM ID" when you only need matching VM IDs.
// -//
// -// Template Parameters:
// -//   - Filter: raw SQL WHERE expression from the filter parser (optional)
// -//   - Limit:  max results, 0 = unlimited (optional)
// -//   - Offset: skip first N results (optional)
// -// */ -}}
// -// SELECT DISTINCT i."VM ID" AS id
// -// FROM vinfo i
// -// LEFT JOIN vcpu c ON i."VM ID" = c."VM ID"
// -// LEFT JOIN vmemory m ON i."VM ID" = m."VM ID"
// -// LEFT JOIN vdisk dk ON i."VM ID" = dk."VM ID"
// -// LEFT JOIN vdatastore ds ON ds."Name" = regexp_extract(COALESCE(dk."Path", dk."Disk Path"), '\[([^\]]+)\]', 1)
// -// LEFT JOIN vnetwork n ON i."VM ID" = n."VM ID"
// -// LEFT JOIN concerns con ON i."VM ID" = con."VM_ID"
// -// WHERE 1=1
// -// {{- if .Filter }} AND {{ .Filter }}{{ end }}
// -// {{- if and .Limit (gt .Limit 0) }} LIMIT {{ .Limit }}{{ end }}
// -// {{- if and .Offset (gt .Offset 0) }} OFFSET {{ .Offset }}{{ end }};
package filter
