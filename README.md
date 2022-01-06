# Dom's Data Build Tool

[![Build Status](https://github.com/monzo/ddbt/actions/workflows/tests.yml/badge.svg)](https://github.com/monzo/ddbt/actions/workflows/tests.yml)
[![GoDoc](https://godoc.org/github.com/monzo/ddbt?status.svg)](https://godoc.org/github.com/monzo/ddbt)

This repo represents my attempt to build a fast version of [DBT](https://www.getdbt.com/) which gets very slow on large 
projects (3000+ data models). This project attempts to be a direct drop in replacement for DBT at the command line.

*Warning:* This is experimental and may not work exactly as you expect

## Installation
1. Clone this repo
```bash
$ git clone git@github.com:monzo/ddbt.git
```

2. Change directory into cloned repo
```bash
$ cd ddbt
```

3. Install (requires go-lang)
```bash
$ go install
```

4. Confirm installation
```bash
$ ddbt --version
ddbt version 0.6.3
```

## Command Quickstart
- `ddbt run` will compile and execute all your models, or those filtered for, against your data warehouse
- `ddbt test` will run all tests referencing all your models, or those filtered for, in your project against your data warehouse
- `ddbt show my_model` will output the compiled SQL to the terminal
- `ddbt copy my_model` will copy the compiled SQL into your clipboard
- `ddbt show-dag` will output the order of how the models will execute
- `ddbt watch` will get act like `run`, followed by `test`. DDBT will then watch your file system for any changes and automatically rerun those parts of the DAG and affected downstream tests or failing tests.
- `ddbt watch --skip-run` is the same as watch, but will skip the initial run (preventing you having to wait for all the models to run) before running the tests and starting to watch your file system.
- `ddbt completion zsh` will generate a shell completion script zsh (or bash if you pass that as argument). Detailed steps to set up the completion script can be found in `ddbt completion --help`
- `ddbt isolate-dag` will create a temporary directory and symlink in all files needed for the given _model_filter_ such that Fishtown's DBT could be run against it without having to be run against every model in your data warehouse
- `ddbt schema-gen -m my_model` will output a new or updated schema yml file for the model provided in the same directory as the dbt model file.
- `ddbt lookml-gen my_model` will generate lookml view and copy it to your clipboard

### Global Arguments
- `--models model_filter` _or_ `-m model_filter`: Instead of running for every model in your project, DDBT will only execute against the requested models. See filters below for what is accepted for `my_model`
- `--threads=n`: force DDBT to run with `n`  threads instead of what is defined in your `dbt_project.yml`
- `--target=x` _or_ `-t x`: force DDBT to run against the `x` output defined in your `profile.yml` instead of the default defined in that file.
- `--upstream=y` _or_ `-u y`: For any references to models outside the explicit models specified by run or test, the upstream target used to read that data will be swapped to `y` instead of the output target of `x`  
- `--fail-on-not-found=false` _or_ `-f=false`: By default, ddbt will fail if a the specified models don't exist, passing in this argument as false will warn instead of failing  
- `--enable-schema-based-tests` _or_ `-s=true`: Schema-based tests are disabled by default for now, but as a way to enable them pass this argument as true 

### Model Filters
When running or testing the project, you may only want to run for a subset of your models.

Currently DDBT supports the following syntax options:
- `-m my_model`: DDBT will only execute against the model with that name
- `-m +my_model`: DDBT will run against `my_model` and all upstreams referenced by it
- `-m my_model+`: DDBT will run against `my_model` and all downstreams that referenced it
- `-m +my_model+`: DDBT will run against `my_model` and both all upstreams and downstreams.
- `-m tag:tagValue`: DDBT will only execute models which have a tag which is equal to `tagValue`
