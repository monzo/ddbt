package bigquery

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/fs"
)

func Run(f *fs.File) error {
	buildFinalQuery(f)

	return errors.New(fmt.Sprintf(
		"Unable to run %s as BigQuery interface not implemented",
		f.Name,
	))
}

func buildFinalQuery(f *fs.File) string {
	var builder strings.Builder

	if udf := f.GetConfig("udf"); udf.Type() == compilerInterface.StringVal {
		builder.WriteString(udf.StringValue)
	}

	// Add a CREATE TABLE wrapper
	builder.WriteString("CREATE OR REPLACE TABLE `")
	builder.WriteString(config.GlobalCfg.Target.ProjectID)
	builder.WriteString("`.`")
	builder.WriteString(config.GlobalCfg.Target.DataSet)
	builder.WriteString("`.`")
	builder.WriteString(f.Name)
	builder.WriteString("` OPTIONS(\n")
	builder.WriteString("\tdescription='Built via DDBT at " + time.Now().Format("2006-01-02 15:04:05 -0700 MST") + "'\n")
	builder.WriteString(") AS (\n")

	// Add the compiled SQL
	builder.WriteString(f.CompiledContents)

	// End the CREATE TABLE wrap
	builder.WriteString(");")

	return builder.String()
}
