# MSSQL Check
A SQL Server plugin for Nagios written in Go.

## Overview
A simple Nagios plugin to execute arbitrary SQL scripts against an instance.

### Features

- Support for SQLCMD style variable substitutions
- Support for "timing" mode to measure duration for any query (useful for canary style queries)
- Support for all standard plugin features:
  - Service Status
  - Detailed Service Status
  - Optional Performance Data
  - Service Status Code

## SQL Scripts
In order to use a SQL script file with this plugin, it must return three result sets:

1. Service status, human readable data: The first row will be included in outgoing alerts and be displayed on summary screen in Nagios. Any additional rows in the first result set will be displayed when viewing the details of a specific service check.
1. Performance data: For some checks you may want to include performance data of some kind that will be written to an external system or used to generate graphs. This result set should be in the form of `Key,Value`, each row being a separate key/value pair. If you are not going to return performance data, simply add `SELECT 'Key',NULL AS [Value]` as the second query in your script.
1. Exit code: Returns a single integer representing the service status (0 - OK, 1 - Warning, 2 - Critical, 3 - Unknown)

### Simple Example
```
// Service Status
SELECT TOP (1) name AS HR  FROM sys.databases
UNION ALL
SELECT name AS HR FROM sys.databases

// Performance Data
SELECT 'Metric' AS Metric, 10 AS Value

// Exit Code
SELECT 0 AS exit_code
```

