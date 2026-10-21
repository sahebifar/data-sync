package handlers

import "sort"

func CompareUserMaps(
	postgresUsers map[string]UserRecord,
	yugabyteUsers map[string]UserRecord,
) []UserDifference {
	emailSet := make(map[string]struct{})

	for email := range postgresUsers {
		emailSet[email] = struct{}{}
	}

	for email := range yugabyteUsers {
		emailSet[email] = struct{}{}
	}

	emails := make([]string, 0, len(emailSet))
	for email := range emailSet {
		emails = append(emails, email)
	}
	sort.Strings(emails)

	differences := make([]UserDifference, 0)

	for _, email := range emails {
		pgUser, inPostgres := postgresUsers[email]
		ybUser, inYugabyte := yugabyteUsers[email]

		switch {
		case inPostgres && !inYugabyte:
			differences = append(differences, UserDifference{
				Email:        email,
				Status:       "missing_in_yugabyte",
				PostgresUser: &pgUser,
			})

		case !inPostgres && inYugabyte:
			differences = append(differences, UserDifference{
				Email:        email,
				Status:       "missing_in_postgres",
				YugabyteUser: &ybUser,
			})

		default:
			fields := make([]string, 0)

			if pgUser.FirstName != ybUser.FirstName {
				fields = append(fields, "first_name")
			}

			if pgUser.LastName != ybUser.LastName {
				fields = append(fields, "last_name")
			}

			if !sameAge(pgUser.Age, ybUser.Age) {
				fields = append(fields, "age")
			}

			if len(fields) > 0 {
				differences = append(differences, UserDifference{
					Email:           email,
					Status:          "data_mismatch",
					PostgresUser:    &pgUser,
					YugabyteUser:    &ybUser,
					DifferentFields: fields,
				})
			}
		}
	}

	return differences
}
