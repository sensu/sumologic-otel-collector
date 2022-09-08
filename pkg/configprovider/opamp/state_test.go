package opamp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_stateManager_StatePath(t *testing.T) {
	type fields struct {
		statePath string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "returns the default state path when field is empty",
			want: func() string {
				return filepath.Join(os.TempDir(), "ot-opamp-state")
			}(),
		},
		{
			name: "returns the field state path when field is set",
			fields: fields{
				statePath: "/tmp/foo",
			},
			want: "/tmp/foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &stateManager{
				statePath: tt.fields.statePath,
			}
			if got := m.StatePath(); got != tt.want {
				t.Errorf("stateManager.StatePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stateManager_Load(t *testing.T) {
	type fields struct {
		statePath string
	}
	tests := []struct {
		name       string
		fields     fields
		beforeHook func(*testing.T, *stateManager)
		want       *agentState
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "returns error when read file fails",
			fields: fields{
				statePath: "/foo/bar/fake/path",
			},
			wantErr:    true,
			wantErrMsg: "open /foo/bar/fake/path: no such file or directory",
		},
		{
			name: "returns error when json unmarshaling fails",
			fields: fields{
				statePath: filepath.Join(t.TempDir(), "ot-opamp-state-test"),
			},
			beforeHook: func(t *testing.T, m *stateManager) {
				f, err := os.OpenFile(m.StatePath(), os.O_RDWR|os.O_CREATE, 0755)
				require.NoError(t, err)
				t.Cleanup(func() {
					f.Close()
				})

				_, err = f.Write([]byte("{42"))
				require.NoError(t, err)
			},
			wantErr:    true,
			wantErrMsg: "invalid character '4' looking for beginning of object key string",
		},
		{
			name: "returns error when validation fails",
			fields: fields{
				statePath: filepath.Join(t.TempDir(), "ot-opamp-state-test"),
			},
			beforeHook: func(t *testing.T, m *stateManager) {
				f, err := os.OpenFile(m.StatePath(), os.O_RDWR|os.O_CREATE, 0755)
				require.NoError(t, err)
				t.Cleanup(func() {
					f.Close()
				})

				state := agentState{}
				bytes, err := json.Marshal(state)
				require.NoError(t, err)

				_, err = f.Write(bytes)
				require.NoError(t, err)
			},
			wantErr:    true,
			wantErrMsg: "instance id is empty",
		},
		{
			name: "loads state without errors",
			fields: fields{
				statePath: filepath.Join(t.TempDir(), "ot-opamp-state-test"),
			},
			beforeHook: func(t *testing.T, m *stateManager) {
				state := &agentState{
					InstanceId: "foo",
				}
				require.NoError(t, m.Save(state))
			},
			want: &agentState{
				InstanceId: "foo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &stateManager{
				statePath: tt.fields.statePath,
			}
			if tt.beforeHook != nil {
				tt.beforeHook(t, m)
			}
			got, err := m.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("stateManager.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.wantErrMsg {
				t.Errorf("stateManager.Load() error msg = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stateManager.Load() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stateManager_Save(t *testing.T) {
	type fields struct {
		statePath string
	}
	type args struct {
		state *agentState
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "returns error when validation fails",
			fields: fields{
				statePath: filepath.Join(t.TempDir(), "ot-opamp-state-test"),
			},
			args: args{
				state: &agentState{},
			},
			wantErr:    true,
			wantErrMsg: "instance id is empty",
		},
		{
			name: "returns error when open file fails",
			fields: fields{
				statePath: "/foo/bar/fake/path",
			},
			args: args{
				state: &agentState{
					InstanceId: newInstanceId(),
				},
			},
			wantErr:    true,
			wantErrMsg: "open /foo/bar/fake/path: no such file or directory",
		},
		{
			name: "saves state without errors",
			args: args{
				state: &agentState{
					InstanceId: newInstanceId(),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &stateManager{
				statePath: tt.fields.statePath,
			}
			err := m.Save(tt.args.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("stateManager.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.wantErrMsg {
				t.Errorf("stateManager.Save() error msg = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}
