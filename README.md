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
source: # Source FTP server
  address: ftp.server1:21
  username: username
  password: password

destination: # destination FTP server
  address: ftp.server2:21
  username: username
  password: password

# list of files or directories to be copied from source to destination on FXP mode
# directories should ends with '/'
files: 
  - upload/ # directory
  - upload2/hello.txt # file
```

