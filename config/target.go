package config

import "math/rand"

type Target struct {
	Name                 string
	ProjectID            string
	DataSet              string
	Location             string
	Threads              int
	ProjectSubstitutions map[string]map[string]string
	ExecutionProjects    []string

	ReadUpstream *Target // If the reference is outside this DAG, this is the target we should read from
}

func (t *Target) Copy() *Target {
	projectSubs := make(map[string]map[string]string)
	for tag, sub := range t.ProjectSubstitutions {
		projectSubs[tag] = make(map[string]string)

		for sourceProject, targetProject := range sub {
			projectSubs[tag][sourceProject] = targetProject
		}
	}

	executionProjects := make([]string, len(t.ExecutionProjects))
	for i, project := range t.ExecutionProjects {
		executionProjects[i] = project
	}

	var defaultUpstream *Target
	if t.ReadUpstream != nil {
		defaultUpstream = t.ReadUpstream.Copy()
	}

	return &Target{
		Name:                 t.Name,
		ProjectID:            t.ProjectID,
		DataSet:              t.DataSet,
		Location:             t.Location,
		Threads:              t.Threads,
		ProjectSubstitutions: projectSubs,
		ExecutionProjects:    executionProjects,
		ReadUpstream:         defaultUpstream,
	}
}

func (t *Target) RandExecutionProject() string {
	if len(t.ExecutionProjects) == 0 {
		return t.ProjectID
	}

	i := rand.Intn(len(t.ExecutionProjects))
	return t.ExecutionProjects[i]
}
