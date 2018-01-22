package build

var emptyEnv []string

// func TestRun(t *testing.T) {
// 	assert.NoError(t, Run(
// 		context.New(config.Project{}),
// 		buildtarget.Runtime,
// 		[]string{"go", "list", "./..."},
// 		emptyEnv,
// 	))
// }

// func TestRunInvalidCommand(t *testing.T) {
// 	assert.Error(t, Run(
// 		context.New(config.Project{}),
// 		buildtarget.Runtime,
// 		[]string{"gggggo", "nope"},
// 		emptyEnv,
// 	))
// }
