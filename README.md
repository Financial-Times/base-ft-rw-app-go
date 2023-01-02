# Base app for Read/Write apps in Go

TODO add lots more info!

## Steps to create a new read/write app

* Copy an existing app
* Rename the puppet folder to a new class name, e.g. ft-org_rw_nweo4j
* Go edit all the files in the manifests folder to match that change, using e.g. org_rw_neo4j
* Similarly with the Modulefile
* Add the exe file into .gitignore
* Amend main.go to specify the details of the command line args
* Implement Service for your service
* Add a model for your service
* Add a healthcheck for your service
* Done!
test
