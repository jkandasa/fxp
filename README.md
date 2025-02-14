### FXP
This tool transfers the files between two FTP servers with FXP

### Build from source
```sh
./build.sh
```

#### Run the utility
```sh
$ ./fxp -h
Usage of ./fxp:
  -config string
        config.yaml file location (default "./config.yaml")
  -version
        Prints this tool version
```
#### Example
```sh
$ ./fxp -config ./config.yaml
```


#### Config file
```yaml
# general config
is_mdtm: false
connection_timeout: 5s
fxp_transfer_timeout: 10m

source: # Source FTP server
  address: ftp.server1:21
  is_tls: false
  insecure: false
  username: username
  password: password
  debug: true # prints ftp communication details

destination: # destination FTP server
  address: ftp.server2:21
  is_tls: false
  insecure: false
  username: username
  password: password
  debug: true

# list of files or directories to be copied from source to destination on FXP mode
# directories should ends with '/'
files: 
  - upload/ # directory
  - upload2/hello.txt # file
```

