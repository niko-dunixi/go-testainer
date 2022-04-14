package postgres

// var (
// 	// globalClientOnce sync.Once
// 	// globalClient     Testainer = nil
// )

type Config struct {
	Version                      string
	Database, Username, Password string
}

type postgresTestainer struct {
	// t testainer.Testainer
}

// func New() (testainer.Testainer, error) {
// 	t, err := testainer.New()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return postgresTestainer{
// 		t: t,
// 	}, nil
// }

// func (pt postgresTestainer) Use(ctx context.Context, config testainer.Config, callback testainer.CallbackFunc) error {

// }

// func (pt postgresTestainer) Run(ctx context.Context, config testainer.Config) (testainer.ContainerDetails, testainer.CleanupFunc, error) {

// }

// func (pc PostgresConfig) String() string {
// 	return fmt.Sprintf(
// 		"host=%s port=%d user=%s password=%s dbname=%s",
// 		pc.Host, pc.Port, pc.Username, pc.Password, pc.DB,
// 	)
// }

// // Creates Docker centric environment tuple strings slice.
// // Each element is `KEY=VALUE`
// func (pc PostgresConfig) DockerEnv() []string {
// 	// Reference https://hub.docker.com/_/postgres
// 	values := make([]string, 0, 10)
// 	if pc.DB != "" {
// 		values = append(values, fmt.Sprintf("POSTGRES_DB=%s", pc.DB))
// 	}
// 	if pc.Username != "" {
// 		values = append(values, fmt.Sprintf("POSTGRES_USER=%s", pc.Username))
// 	}
// 	if pc.Password != "" {
// 		values = append(values, fmt.Sprintf("POSTGRES_PASSWORD=%s", pc.Password))
// 	}
// 	return values
// }

// type PostgresStringConsumer func(c string) error

// func UsePostgres(t *testing.T, ctx context.Context, config PostgresConfig, callback func(string) error) error {
// 	return nil
// }

// func GetPostgres(t *testing.T, ctx context.Context, config PostgresConfig) {
// 	t.Helper()
// 	containerConfig, hostConfig, err := createContainerConfig(config)
// 	if err != nil {
// 		t.Fatalf("could not create configuration for postgres: %v", err)
// 	}

// }

// func createContainerConfig(c PostgresConfig) (container.Config, container.HostConfig, error) {
// 	hostPort, err := freeport.GetFreePort()
// 	if err != nil {
// 		return container.Config{}, container.HostConfig{}, fmt.Errorf("couldn't find free host port: %w", err)
// 	}
// 	containerPort, err := nat.NewPort("tcp", "5432")
// 	if err != nil {
// 		return container.Config{}, container.HostConfig{}, fmt.Errorf("couldn't create cointainer port: %w", err)
// 	}
// 	containerConfig := container.Config{
// 		Image: "docker.io/postgres:14",
// 		Env:   c.DockerEnv(),
// 	}
// 	hostConfig := container.HostConfig{
// 		PortBindings: nat.PortMap{
// 			containerPort: []nat.PortBinding{
// 				{
// 					HostIP:   "0.0.0.0", // Should this be bound to the loopback address instead?
// 					HostPort: strconv.Itoa(hostPort),
// 				},
// 			},
// 		},
// 	}
// 	return containerConfig, hostConfig, nil
// }

// // func UsePostgres(t *testing.T, ctx context.Context, config Config, callback func(string) error) error {
// // 	t.Helper()
// // 	connection, cleanup, err := StartPostgres(t, ctx, config)
// // 	if err != nil {
// // 		return err
// // 	}
// // 	defer cleanup()
// // 	return callback(connection)
// // }

// // func StartPostgres(t *testing.T, ctx context.Context, config Config) (connection string, cleanup func() error, err error) {
// // 	t.Helper()
// // 	docker, err := dkr.NewClientWithOpts(dkr.FromEnv)
// // 	connection = config.String()
// // 	containerConfig := container.Config{
// // 		Image: "docker.io/postgres:14",
// // 		Env: []string{
// // 			"POSTGRES_DB=postgres",
// // 			"POSTGRES_USER=postgres",
// // 			"POSTGRES_PASSWORD=postgres",
// // 		},
// // 	}
// // 	hostPort, _ := freeport.GetFreePort()
// // 	containerPort, err := nat.NewPort("tcp", "5432")
// // 	if err != nil {
// // 		return "", nil, fmt.Errorf("this should not happen, but we couldn't create a nat port: %w", err)
// // 	}
// // 	hostConfig := container.HostConfig{
// // 		PortBindings: nat.PortMap{
// // 			containerPort: []nat.PortBinding{
// // 				{
// // 					HostIP:   "0.0.0.0", // Should this be bound to the loopback address instead?
// // 					HostPort: strconv.Itoa(hostPort),
// // 				},
// // 			},
// // 		},
// // 	}

// // 	container, err := docker.ContainerCreate(
// // 		ctx, &containerConfig, &hostConfig, nil, nil,
// // 		fmt.Sprintf("postgres-%d_%d", hostPort, time.Now().Unix()),
// // 	)
// // 	if err != nil {
// // 		return "", nil, fmt.Errorf("this should not happen, but we couldn't create a nat port: %w", err)
// // 	}
// // 	dockerCleanup := func() {
// // 		t.Logf("Cleaning up docker container: %s", container.ID)
// // 		timeout := time.Duration(time.Second * 5)
// // 		if err := docker.ContainerStop(ctx, container.ID, &timeout); err != nil {
// // 			t.Fatalf("Cleaning up docker container: %s", err)
// // 		}
// // 		err := docker.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
// // 			RemoveVolumes: true,
// // 			Force:         true,
// // 		})
// // 		if err != nil {
// // 			t.Logf("could not remove container: %+v", err)
// // 		}
// // 	}
// // 	docker, err := dkr.NewClientWithOpts(dkr.FromEnv)

// // 	defer func() {
// // 		// Ensure that if we fail anywhere along the way to our initialization, we clean up the container
// // 		// otherwise we will pass the cleanup function back to the caller to handle themselves as part
// // 		// of the testing lifecycle
// // 		if err := recover(); err != nil {
// // 			defer dockerCleanup()
// // 			panic(err)
// // 		}
// // 	}()

// // }
