package bigquery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"ddbt/compilerInterface"
	"ddbt/config"
	"ddbt/fs"
)

var (
	clientsMutex sync.Mutex
	clients      map[string]*bigquery.Client
)

func Init(cfg *config.Config) (err error) {
	clients = make(map[string]*bigquery.Client)

	if _, err := GetClientFor(cfg.Target.ProjectID); err != nil {
		return err
	}

	return nil
}

func GetClientFor(project string) (*bigquery.Client, error) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	client, found := clients[project]
	if !found {
		var err error
		client, err = bigquery.NewClient(context.Background(), project)

		if err != nil {
			return nil, err
		}

		clients[project] = client
	}

	return client, nil
}

func Run(ctx context.Context, f *fs.File) (string, error) {
	query := BuildQuery(f)

	if strings.TrimSpace(query) == "" {
		return "", nil
	}

	target, err := f.GetTarget()
	if err != nil {
		return "", err
	}

	switch {
	case target.ProjectID == "":
		return "", errors.New("no project ID defined to run query against")
	case target.DataSet == "":
		return "", errors.New("no dataset defined to run query against")
	}

	client, err := GetClientFor(target.RandExecutionProject())
	if err != nil {
		return "", err
	}

	dataset := client.DatasetInProject(target.ProjectID, target.DataSet)

	q := client.Query(query)
	q.Location = target.Location

	// Default read information
	q.DefaultProjectID = target.ProjectID
	q.DefaultDatasetID = target.DataSet
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

type Value = bigquery.Value
type Schema = bigquery.Schema

func Quote(value string) string {
	// ' => \'
	// \' => \\\'
	// \ => \\
	return strings.Replace(
		strings.Replace(
			value,
			"\\",
			"\\\\'",
			-1,
		),
		"'",
		"\\'",
		-1,
	)
}

func NumberRows(query string, target *config.Target) (uint64, error) {
	ctx := context.Background()

	switch {
	case target.ProjectID == "":
		return 0, errors.New("no project ID defined to run query against")
	case target.DataSet == "":
		return 0, errors.New("no dataset defined to run query against")
	}

	client, err := GetClientFor(target.RandExecutionProject())
	if err != nil {
		return 0, err
	}

	q := client.Query(query)
	q.Location = target.Location

	// Default read information
	q.DefaultProjectID = target.ProjectID
	q.DefaultDatasetID = target.DataSet

	job, err := q.Run(ctx)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Unable to run query: %s", err))
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Error executing: %s", err))
	}

	if status.State != bigquery.Done {
		return 0, errors.New(fmt.Sprintf("Exuection job %s in state %s", job.ID(), status.State))
	}

	if err := status.Err(); err != nil {
		return 0, errors.New(fmt.Sprintf("Job result in an error: %s", err))
	}

	itr, err := job.Read(ctx)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Job result in an error: %s", err))
	}

	return itr.TotalRows, nil
}

func GetRows(query string, target *config.Target) ([][]Value, Schema, error) {
	ctx := context.Background()

	switch {
	case target.ProjectID == "":
		return nil, nil, errors.New("no project ID defined to run query against")
	case target.DataSet == "":
		return nil, nil, errors.New("no dataset defined to run query against")
	}

	client, err := GetClientFor(target.RandExecutionProject())
	if err != nil {
		return nil, nil, err
	}

	q := client.Query(query)
	q.Location = target.Location

	// Default read information
	q.DefaultProjectID = target.ProjectID
	q.DefaultDatasetID = target.DataSet

	job, err := q.Run(ctx)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Unable to run query %s\n\n%s", query, err))
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Error executing query %s\n\n%s", query, err))
	}

	if status.State != bigquery.Done {
		return nil, nil, errors.New(fmt.Sprintf("Model %s's exuection job %s in state %s", query, job.ID(), status.State))
	}

	if err := status.Err(); err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Model %s's job result in an error: %s", query, err))
	}

	itr, err := job.Read(ctx)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("Model %s's job result in an error: %s", query, err))
	}

	rows := make([][]Value, 0)
	schema := itr.Schema

	for {
		var row []bigquery.Value
		err := itr.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		rows = append(rows, row)
	}

	return rows, schema, nil
}

func GetColumnsFromTable(table string, target *config.Target) (Schema, error) {
	_, schema, err := GetRows(fmt.Sprintf("SELECT * FROM %s LIMIT 0", table), target)
	if err != nil {
		return nil, err
	}

	return schema, nil
}
