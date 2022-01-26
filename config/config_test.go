package config

import "testing"

func Test_handleCustomConfigPath(t *testing.T) {
	type args struct {
		customConfigPath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// path with trailing slash returns same path
		{
			name: "path with trailing slash",
			args: args{
				customConfigPath: "foo/",
			},
			want:    "foo/",
			wantErr: false,
		},
		// path without trailing slash returns path with trailing slash
		{
			name: "path without trailing slash",
			args: args{
				customConfigPath: "foo",
			},
			want:    "foo/",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handleCustomConfigPath(tt.args.customConfigPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleCustomConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("handleCustomConfigPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
