package main

import "testing"

func Test_checkIfPathIsRelative(t *testing.T) {
	type args struct {
		string path
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkIfPathIsRelative(tt.args.string)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkIfPathIsRelative() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkIfPathIsRelative() got = %v, want %v", got, tt.want)
			}
		})
	}
}
