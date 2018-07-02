#!/bin/bash
set -ex
sfctl application delete --application-id testapp_java
sfctl application unprovision --application-type-name testapp_javaType --application-type-version 1.0.0
sfctl store delete --content-path testapp_java
