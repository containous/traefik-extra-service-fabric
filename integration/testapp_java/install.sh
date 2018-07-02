#!/bin/bash
set -ex
sfctl application upload --path testapp_java --show-progress
sfctl application provision --application-type-build-path testapp_java
sfctl application create --app-name fabric:/testapp_java --app-type testapp_javaType --app-version 1.0.0
