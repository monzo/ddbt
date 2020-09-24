# Dom's Data Build Tool

This repo represents my attempt to build a fast version of [DBT](https://www.getdbt.com/) which gets very slow on large 
projects (3000+ data models). This project attempts to be a direct drop in replacement for DBT at the command line.

*Warning:* This is experimental and may not work exactly as you expect

## Command Quickstart

- `ddbt run` will compile and execute all your models, or those filtered for, against your data warehouse
- `ddbt test` will run all tests referencing all your models, or those filtered for, in your project against your data warehouse
- `ddbt show my_model` will output the compiled SQL to the terminal
- `ddbt copy my_model` will copy the compiled SQL into your clipboard

### Global Arguments
- `--models model_filter` _or_ `-m model_filter`: Instead of running for every model in your project, DDBT will only execute against the requested models. See filters below for what is accepted for `my_model`
- `--threads=n`: force DDBT to run with `n`  threads instead of what is defined in your `dbt_project.yml`
- `--target=x` _or_ `-t x`: force DDBT to run against the `x` output defined in your `profile.yml` instead of the default defined in that file.
- `--upstream=y` _or_ `-u y`: For any references to models outside the explicit models specified by run or test, the upstream target used to read that data will be swapped to `y` instead of the output target of `x`  

### Model Filters
When running or testing the project, you may only want to run for a subset of your models.

Currently DDBT supports the following syntax options:
- `-m my_model`: DDBT will only execute against the model with that name
- `-m +my_model`: DDBT will run against `my_model` and all upstreams referenced by it
- `-m my_model+`: DDBT will run against `my_model` and all downstreams that referenced it
- `-m +my_model+`: DDBT will run against `my_model` and both all upstreams and downstreams.