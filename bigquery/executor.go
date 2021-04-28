package bigquery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
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
		return query, errors.New(fmt.Sprintf("Model %s's execution job %s in state %d", f.Name, job.ID(), status.State))
	}

	if err := status.Err(); err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Model %s's job result in an error: %s", f.Name, err))
	}

	return query, nil
}

func RunQuery(ctx context.Context, modelName string, query string, target *config.Target) (string, error) {

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
	q.Dst = dataset.Table(modelName)
	q.CreateDisposition = bigquery.CreateIfNeeded
	q.WriteDisposition = bigquery.WriteTruncate

	job, err := q.Run(ctx)
	if err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Unable to run model %s: %s", modelName, err))
	}

	status, err := job.Wait(ctx)
	if err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Error executing model %s: %s", modelName, err))
	}

	if status.State != bigquery.Done {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Model %s's execution job %s in state %d", modelName, job.ID(), status.State))
	}

	if err := status.Err(); err != nil {
		if err == context.Canceled {
			return "", err
		}
		return query, errors.New(fmt.Sprintf("Model %s's job result in an error: %s", modelName, err))
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

func ValueAsUint64(value Value) (uint64, error) {
	switch v := value.(type) {
	case int:
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case int32:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float32:
		return uint64(v), nil
	case float64:
		return uint64(v), nil
	default:
		return 0, errors.New(fmt.Sprintf("unable to convert %v into a uint64", reflect.TypeOf(value)))
	}
}

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
		return 0, errors.New(fmt.Sprintf("Execution job %s in state %d", job.ID(), status.State))
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

func GetRows(ctx context.Context, query string, target *config.Target) ([][]Value, Schema, error) {
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
		return nil, nil, errors.New(fmt.Sprintf("Model %s's execution job %s in state %d", query, job.ID(), status.State))
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

// GetColumnsFromTable is a fallback GetColumnsFromTableWithContext
// with a background context.
func GetColumnsFromTable(table string, target *config.Target) (Schema, error) {
	return GetColumnsFromTableWithContext(context.Background(), table, target)
}

func GetColumnsFromTableWithContext(ctx context.Context, table string, target *config.Target) (Schema, error) {
	_, schema, err := GetRows(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT 0", table), target)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

// LoadSeedFile loads a CSV file to BigQuery as a table.
func LoadSeedFile(ctx context.Context, seed *fs.SeedFile) error {
	target, err := seed.GetTarget()
	if err != nil {
		return err
	}

	switch {
	case target.ProjectID == "":
		return errors.New("no project ID defined to run query against")
	case target.DataSet == "":
		return errors.New("no dataset defined to run query against")
	}

	client, err := GetClientFor(target.RandExecutionProject())
	if err != nil {
		return err
	}

	f, err := os.Open(seed.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	rs := bigquery.NewReaderSource(f)
	rs.AllowJaggedRows = false
	rs.SkipLeadingRows = 1
	rs.SourceFormat = bigquery.CSV

	if seed.HasSchema() {
		schema, err := getSeedSchema(seed)
		if err != nil {
			return err
		}
		rs.Schema = schema
		rs.AutoDetect = false
	} else {
		rs.AutoDetect = true
	}

	dataset := client.DatasetInProject(target.ProjectID, target.DataSet)
	loader := dataset.Table(seed.Name).LoaderFrom(rs)
	loader.WriteDisposition = bigquery.WriteTruncate // Replace table content

	job, err := loader.Run(ctx)
	if err != nil {
		return fmt.Errorf("Unable to start load job for file: %s: %w", seed.Path, err)
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("Error loading seed file %s: %w", seed.Path, err)
	}
	if status.State != bigquery.Done {
		return fmt.Errorf("Seed file %s's loading job %s in state %d", seed.Path, job.ID(), status.State)
	}
	if err := status.Err(); err != nil {
		return fmt.Errorf("Seed file %s's loading job has an error: %w", seed.Path, err)
	}

	return nil
}

func getSeedSchema(seed *fs.SeedFile) (bigquery.Schema, error) {
	schema := make([]*bigquery.FieldSchema, 0, len(seed.Columns))
	// Use schema specified column types if available
	for _, column := range seed.Columns {
		schema = append(schema, &bigquery.FieldSchema{
			Name: column,
			Type: bigquery.FieldType(strings.ToUpper(seed.ColumnTypes[column])),
		})
	}
	return schema, nil
}
