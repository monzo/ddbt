# Dom's Data Build Tool

This repo represents my attempt to build a fast version of [DBT](https://www.getdbt.com/) which gets very slow on large 
projects (3000+ data models). This project attempts to be a direct drop in replacement for DBT at the command line.

*Warning:* This is experimental and may not work exactly as you expect

## Command Quickstart

- `ddbt run` will compile and execute all your models against your data warehouse
- `ddbt run -m my_model` will compile all your models, but only execute `my_model` in your data warehouse
- `ddbt test` will run all tests in your project against your data warehouse
- `ddbt test -m my_model` will run all tests which reference your model, but not run the model itself.
- `ddbt show my_model` will output the compiled SQL to the terminal
- `ddbt copy my_model` will copy the compiled SQL into your clipboard

### Arguments
- `--threads=n` force DDBT to run with `n`  threads instead of what is defined in your `dbt_project.yml`
- `--target=x`  force DDBT to run against the `x` output defined in your `profile.yml` instead of the default defined in that file.  