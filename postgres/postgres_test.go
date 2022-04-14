package postgres

// func TestRun(t *testing.T) {
// 	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*15)
// 	defer ctxCancel()

// 	config := testainer.Config{
// 		Registry: testainer.DockerHubLibraryRegistry,
// 		Image:    "postgres",
// 		Tag:      "14",
// 		Env: map[string]string{
// 			"POSTGRES_DB":       "mydb",
// 			"POSTGRES_USER":     "paulfreaknbaker",
// 			"POSTGRES_PASSWORD": "megasecure678",
// 		},
// 		Port: 5432,
// 	}
// 	wasCalled := false
// 	err := testainer.Use(ctx, config, func(ctx context.Context, containerDetails testainer.ContainerDetails) error {
// 		wasCalled = true
// 		return nil
// 	})
// 	assert.NoError(t, err)
// 	assert.True(t, wasCalled, "callback was not executed")
// }
