package test

import (
	"bytes"
	"testing"

	"github.com/philslol/kritis3m_scalev2/cmd/cli"
	mockpkg "github.com/philslol/kritis3m_scalev2/cmd/test/mock"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestCreateVersionSet(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedError  bool
		expectedOutput string
		mockResponse   *v1.VersionSetResponse
		mockError      error
	}{
		{
			name: "successful creation",
			args: []string{
				"version-set", "create",
				"--name", "test-version",
				"--description", "test description",
				"--created-by", "test-user",
			},
			expectedError: false,
			mockResponse: &v1.VersionSetResponse{
				VersionSet: &v1.VersionSet{
					Id:          "test-id",
					Name:        "test-version",
					Description: "test description",
					CreatedBy:   "test-user",
				},
			},
			mockError: nil,
		},
		{
			name: "missing required flag",
			args: []string{
				"version-set", "create",
				"--description", "test description",
			},
			expectedError: true,
			mockResponse:  nil,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mockpkg.MockSouthboundClient)
			if !tt.expectedError {
				mockClient.On("CreateVersionSet", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
			}

			cli.SetTestClient(mockClient)
			cmd := cli.GetRootCommand()
			output, err := executeCommand(cmd, tt.args...)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, "Version set created")
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestReadVersionSet(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedError  bool
		expectedOutput string
		mockResponse   *v1.VersionSetResponse
		mockError      error
	}{
		{
			name: "successful read",
			args: []string{
				"version-set", "read",
				"--id", "test-id",
			},
			expectedError: false,
			mockResponse: &v1.VersionSetResponse{
				VersionSet: &v1.VersionSet{
					Id:          "test-id",
					Name:        "test-version",
					Description: "test description",
					CreatedBy:   "test-user",
				},
			},
			mockError: nil,
		},
		{
			name: "missing id flag",
			args: []string{
				"version-set", "read",
			},
			expectedError: true,
			mockResponse:  nil,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mockpkg.MockSouthboundClient)
			if !tt.expectedError {
				mockClient.On("GetVersionSet", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
			}

			cli.SetTestClient(mockClient)
			cmd := cli.GetRootCommand()
			output, err := executeCommand(cmd, tt.args...)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, "Version set:")
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateVersionSet(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedError  bool
		expectedOutput string
		mockResponse   *emptypb.Empty
		mockError      error
	}{
		{
			name: "successful update",
			args: []string{
				"version-set", "update",
				"--id", "test-id",
				"--name", "updated-version",
				"--description", "updated description",
			},
			expectedError: false,
			mockResponse:  &emptypb.Empty{},
			mockError:     nil,
		},
		{
			name: "missing id flag",
			args: []string{
				"version-set", "update",
				"--name", "updated-version",
			},
			expectedError: true,
			mockResponse:  nil,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mockpkg.MockSouthboundClient)
			if !tt.expectedError {
				mockClient.On("UpdateVersionSet", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
			}

			cli.SetTestClient(mockClient)
			cmd := cli.GetRootCommand()
			output, err := executeCommand(cmd, tt.args...)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, "Version set updated")
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestDeleteVersionSet(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedError  bool
		expectedOutput string
		mockResponse   *emptypb.Empty
		mockError      error
	}{
		{
			name: "successful delete",
			args: []string{
				"version-set", "delete",
				"--id", "test-id",
			},
			expectedError: false,
			mockResponse:  &emptypb.Empty{},
			mockError:     nil,
		},
		{
			name: "missing id flag",
			args: []string{
				"version-set", "delete",
			},
			expectedError: true,
			mockResponse:  nil,
			mockError:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mockpkg.MockSouthboundClient)
			if !tt.expectedError {
				mockClient.On("DeleteVersionSet", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
			}

			cli.SetTestClient(mockClient)
			cmd := cli.GetRootCommand()
			output, err := executeCommand(cmd, tt.args...)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, "Version set deleted")
				mockClient.AssertExpectations(t)
			}
		})
	}
}
