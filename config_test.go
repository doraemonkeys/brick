package brick

import "testing"

func TestBrickManager_addConfigFileJson(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent []byte
		wantErr     bool
	}{
		{
			name:        "test1",
			jsonContent: []byte(`{"xxx":"xxx"}`),
			wantErr:     true,
		},
		{
			name:        "test2",
			jsonContent: []byte(`{"bricks":[{"liveID":"test1","config":{"prefix":"test1"}}]}`),
			wantErr:     true,
		},
		{
			name:        "test3",
			jsonContent: []byte(`{"bricks":[]}`),
			wantErr:     false,
		},
		{
			name: "test4",
			jsonContent: []byte(`[
	{
		"MetaData": {
			"Name": "PrintServiceConfig",
			"TypeID": "printService"
		},
		"Lives": [
			{
				"LiveID": "printService",
				"Config": {
					"Prefix": "Custom Prefix"
				}
			}
		]
	}
]`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := brickManager.addConfigFileJson(tt.jsonContent); (err != nil) != tt.wantErr {
				t.Errorf("BrickManager.addConfigFileJson() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
