module fxp

go 1.23.4

require (
	github.com/jlaffaye/ftp v0.2.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
)

// in this fork exposed "conn" and "cmd"
// https://github.com/jkandasa/ftp/commit/29a5125a0d808a0efdf5ca0e6b09dd2738247845
replace github.com/jlaffaye/ftp => github.com/jkandasa/ftp v0.0.0-20250214085635-29a5125a0d80
