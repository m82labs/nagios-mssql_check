package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
    "regexp"

	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	var (
		userid     = flag.String("U", "sa", "User name to connect with")
		password   = flag.String("P", "password!", "User password")
		server     = flag.String("h", "localhost", "server_name[\\instance_name]")
		database   = flag.String("d", "master", "Database name to connect to")
		filepath   = flag.String("s", "test.sql", "File path to SQL script file")
		argument   = flag.String("a", "", "SQLCMD-style arguments: 'Key1:Value1,Key2:Value2'")
		integrated = flag.Bool("i", false, "Enable integrated (Windows) authentication")
		get_timing = flag.Bool("t", false, "Returns timing data ONLY for the query executed, normal query output is omitted.")
		generate_check = flag.Bool("g", false, "Generates a check script template you can use to create Nagios checks.")
		sample_script = `----------------------------------------------------------------------------------
-- Script Name: DatabaseHealthCheck.sql
--
-- Desc: This is a sample script that checks if any databases are in 'suspect'
-- mode and shows the basics of using this plugin.
--
-- If you need to pass in any variables, use SQLCMD style syntax:
--   '$(varname)'
--
-- And then use the '-a' option to pass in a value:
--   mssql_check -s myscript.sql -a 'varname:5,varname2'
--
-- Auth: Mark Wilkinson
-- Date: 2017/12/27
--
-- Change History
-- ----------------
----------------------------------------------------------------------------------

USE master

SET NOCOUNT ON;

DECLARE 
	@ServiceStatus VARCHAR(100) = 'Databases healthy',
	@ReturnCode TINYINT = 0 -- 0 -OK, 1 - WARN, 2- CRITICAL, 3 - UNKNOWN

BEGIN TRY
	IF EXISTS (
		SELECT  1
		FROM	sys.databases
		WHERE	state = 4
	)
	BEGIN
	--== Get human readable data and set exit code
		SET @ServiceStatus = 'Database(s) found in suspect state.';
		SET @ReturnCode = 2;
	END

	--== Select our Service Status data
	SELECT	@ServiceStatus AS ServiceStatus
	UNION ALL
	SELECT	REPLACE(REPLACE('{{db}} - {{state}}','{{db}}',name collate SQL_Latin1_General_CP1_CI_AS),'{{state}}',state_desc)
	FROM	sys.databases

	--== Metric
	SELECT NULL AS Metric, NULL AS Value;
	
	--== Return
	SELECT @ReturnCode AS ExitCode;
	
END TRY
BEGIN CATCH
	--== On error, return the error and a status of UNKNOWN to Nagios
	--== Return Error as Human Readable Text
	SELECT CONCAT('Error:',ERROR_MESSAGE()) AS ServiceStatus

	--== Return null metrics
	SELECT NULL AS metric, NULL AS value

	--== Return UNKNOWN
	SELECT 3 AS ExitCode
END CATCH;`
	)
	flag.Parse()

	if *generate_check {
		fmt.Println(sample_script)
		os.Exit(3)
	}

	// Connection string
	var dsn string
	if *integrated {
		dsn = "server=" + *server + ";database=" + *database
	} else {
		dsn = "server=" + *server + ";user id=" + *userid + ";password=" + *password + ";database=" + *database
	}

	// Open a connection
	db, err := sql.Open("mssql", dsn)
	if err != nil {
		fmt.Println("Cannot connect: ", err.Error())
		os.Exit(3)
	}

	// Test Connection
	err = db.Ping()
	if err != nil {
		fmt.Println("Cannot connect: ", err.Error())
		os.Exit(3)
	}
	defer db.Close()

	// Read the script file
	script, err := ioutil.ReadFile(*filepath)
	if err != nil {
		fmt.Println("Cannot open script file: ", err.Error())
		os.Exit(3)
	}

	cmd := string(script)

	// Parse SQLCMD variables
    // Get a list of replacement tokens, break into token/replacement pairs
    pattern := regexp.MustCompile(`\$\(([a-zA-Z_]+:*\S*)\)`)
    found := pattern.FindAllStringSubmatch(cmd, -1)

    tokens := make(map[string][2]string)

    for _, match := range found {
        // Split on ':' to get default values
        token := strings.Split(match[1], ":")
        if len(token) > 1 {
            tokens[token[0]] = [2]string{match[0], token[1]}
        } else {
            tokens[token[0]] = [2]string{match[0], ""}
        }
    }

    // Get list of arguments
    for _, arg := range strings.Split(argument, ",") {
        carg := strings.Split(arg, ":")
        if cap(carg) == 2 {
            tokens[carg[0]] = [2]string{tokens[carg[0]][0], carg[1]}
        } else {
            fmt.Println("Error parsing argurments. Key:Value pair not found.")
            os.Exit(3)
        }
    }

    // Go through the arugments and update the tag/replacment list
    for _, value := range tokens {
        cmd = strings.Replace(cmd, value[0], value[1], -1)
    }

	// Capture timings
	var start time.Time
	var dur int64
	if *get_timing {
		start = time.Now()
	}

	// Execute the script
	rows, err := db.Query(cmd)
	if err != nil {
		fmt.Println("Failed to execute script: ", err.Error())
		os.Exit(3)
	}
	defer rows.Close()

	var service_status string
	var current_hr string
	var exitcode int = 3 //default to UNKNOWN

	// Calculate total time and set variables
	if *get_timing {
		dur = time.Since(start).Nanoseconds() / 1000000
		service_status = fmt.Sprintf("Response Time: %dms", dur)
		service_status += fmt.Sprintf("|instance_latency_ms=%d", dur)
		fmt.Println(service_status)
		os.Exit(0)
	}

	var get_results = true

	for get_results {
		cols, _ := rows.Columns()

		switch strings.ToLower(cols[0]) {
		case "servicestatus": // Service Status
			for rows.Next() {
				err = rows.Scan(&current_hr)
				if err != nil {
					fmt.Println("Failed to parse results: ", err)
					os.Exit(3)
				}
				service_status += current_hr + "\n"
			}
			if !rows.NextResultSet() {
				get_results = false
			}
		case "metric": //Performance Data
			if cap(cols) == 2 {

				var metric sql.NullString
				var value sql.NullString
				service_status += "|"

				for rows.Next() {
					err = rows.Scan(&metric, &value)
					if err != nil {
						fmt.Println("Failed to gather performance data: ", err)
						os.Exit(3)
					}

					if metric.Valid && value.Valid {
						service_status += fmt.Sprintf("%s=%s;", metric.String, value.String)
					}
				}
			} else {
				fmt.Println("Performance data incorrectly formatted, each result must have two fields, 'metric' and 'value'.")
				os.Exit(3)
			}
			if !rows.NextResultSet() {
				get_results = false
			}
		case "exitcode": // Exit Status
			for rows.Next() {
				err = rows.Scan(&exitcode)
				if err != nil {
					fmt.Println("Error getting exit code: ", err)
					os.Exit(3)
				}
			}
			if !rows.NextResultSet() {
				get_results = false
			}
		default:
			get_results = false
		}
	}

	// Output service status information
	if service_status == "" {
		service_status = "No service status returned. Make sure the query is returning a result set with a 'ServiceStatus' field."
	}
	fmt.Println(service_status)

	// Exit with the exit code
	os.Exit(exitcode)
}
