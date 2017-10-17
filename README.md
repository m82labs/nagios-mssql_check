# MSSQL Check
A SQL Server plugin for Nagios written in Go.

## Overview
A simple Nagios plugin to execute arbitrary SQL scripts against an instance.

### Features

- Support for SQLCMD style variable substitutions
- Support for "timing" mode to measure duration for any query (useful for canary style queries)
- Support for all standard plugin feayures:
  - Service Status
  - Detailed Service Status
  - Optional Performance Data
  - Service Status Code
