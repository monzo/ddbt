package bigquery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/fs"
)

var (
	client  *bigquery.Client
	dataset *bigquery.Dataset
)

func Init(cfg *config.Config) (err error) {
	client, err = bigquery.NewClient(
		context.Background(),
		cfg.Target.ProjectID,
	)

	if err != nil {
		return err
	}

	dataset = client.Dataset(cfg.Target.DataSet)

	return nil
}

func Run(ctx context.Context, f *fs.File) (string, error) {
	query := BuildQuery(f)

	q := client.Query(query)
	q.Location = config.GlobalCfg.Target.Location

	// Default read information
	q.DefaultProjectID = config.GlobalCfg.Target.ProjectID
	q.DefaultDatasetID = config.GlobalCfg.Target.DataSet
	q.DisableQueryCache = true

	// Output write information
	q.Dst = dataset.Table(f.Name)
	q.CreateDisposition = bigquery.CreateIfNeeded
	q.WriteDisposition = bigquery.WriteTruncate

	job, err := q.Run(ctx)
	if err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Unable to run model %s: %s", f.Name, err))
	}

	status, err := job.Wait(ctx)
	if err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Error executing model %s: %s", f.Name, err))
	}

	if status.State != bigquery.Done {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Model %s's exuection job %s in state %s", f.Name, job.ID(), status.State))
	}

	if err := status.Err(); err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Model %s's job result in an error: %s", f.Name, err))
	}

	return query, nil
}

func BuildQuery(f *fs.File) string {
	var builder strings.Builder

	if udf := f.GetConfig("udf"); udf.Type() == compilerInterface.StringVal {
		builder.WriteString(udf.StringValue)
	}

	// Add the compiled SQL
	builder.WriteString(f.CompiledContents)

	return builder.String()
}
