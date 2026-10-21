package handlers

import "testing"

func intPtr(value int) *int {
	return &value
}

func TestCompareUserMaps(t *testing.T) {
	tests := []struct {
		name           string
		postgresUsers  map[string]UserRecord
		yugabyteUsers  map[string]UserRecord
		expectedCount  int
		expectedStatus string
		expectedFields []string
	}{
		{
			name: "user missing in YugabyteDB",
			postgresUsers: map[string]UserRecord{
				"amir@test.com": {
					ID:        1,
					FirstName: "Amir",
					LastName:  "Test",
					Email:     "amir@test.com",
					Age:       intPtr(26),
				},
			},
			yugabyteUsers:  map[string]UserRecord{},
			expectedCount:  1,
			expectedStatus: "missing_in_yugabyte",
		},
		{
			name:          "user missing in PostgreSQL",
			postgresUsers: map[string]UserRecord{},
			yugabyteUsers: map[string]UserRecord{
				"amir@test.com": {
					ID:        1,
					FirstName: "Amir",
					LastName:  "Test",
					Email:     "amir@test.com",
					Age:       intPtr(26),
				},
			},
			expectedCount:  1,
			expectedStatus: "missing_in_postgres",
		},
		{
			name: "different user fields",
			postgresUsers: map[string]UserRecord{
				"amir@test.com": {
					ID:        1,
					FirstName: "Amir",
					LastName:  "Test",
					Email:     "amir@test.com",
					Age:       intPtr(26),
				},
			},
			yugabyteUsers: map[string]UserRecord{
				"amir@test.com": {
					ID:        5,
					FirstName: "WrongName",
					LastName:  "Test",
					Email:     "amir@test.com",
					Age:       intPtr(26),
				},
			},
			expectedCount:  1,
			expectedStatus: "data_mismatch",
			expectedFields: []string{"first_name"},
		},
		{
			name: "identical users",
			postgresUsers: map[string]UserRecord{
				"amir@test.com": {
					ID:        1,
					FirstName: "Amir",
					LastName:  "Test",
					Email:     "amir@test.com",
					Age:       intPtr(26),
				},
			},
			yugabyteUsers: map[string]UserRecord{
				"amir@test.com": {
					ID:        5,
					FirstName: "Amir",
					LastName:  "Test",
					Email:     "amir@test.com",
					Age:       intPtr(26),
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differences := CompareUserMaps(
				tt.postgresUsers,
				tt.yugabyteUsers,
			)

			if len(differences) != tt.expectedCount {
				t.Fatalf(
					"expected %d differences, got %d",
					tt.expectedCount,
					len(differences),
				)
			}

			if tt.expectedCount == 0 {
				return
			}

			if differences[0].Status != tt.expectedStatus {
				t.Errorf(
					"expected status %q, got %q",
					tt.expectedStatus,
					differences[0].Status,
				)
			}

			if len(differences[0].DifferentFields) != len(tt.expectedFields) {
				t.Fatalf(
					"expected fields %v, got %v",
					tt.expectedFields,
					differences[0].DifferentFields,
				)
			}

			for i, field := range tt.expectedFields {
				if differences[0].DifferentFields[i] != field {
					t.Errorf(
						"expected field %q, got %q",
						field,
						differences[0].DifferentFields[i],
					)
				}
			}
		})
	}
}