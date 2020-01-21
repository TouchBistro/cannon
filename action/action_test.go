package action

// TESTS WILL BE RE ADDED PROPERLY IN NEXT PR

// var expandRepoVarTests = []struct {
// 	source   string
// 	repoName string
// 	expected string
// }{
// 	{"container_name: $REPONAME_container", "node-boilerplate", "container_name: node-boilerplate_container"},
// 	{"ENV NODE_ENV development", "loyalty-gateway-serivce", "ENV NODE_ENV development"},
// }

// func TestExpandRepoVar(t *testing.T) {
// 	for _, testCase := range expandRepoVarTests {
// 		result := expandRepoVar(testCase.source, testCase.repoName)

// 		logPrefix := fmt.Sprintf("expandRepoVar(%s, %s):", testCase.source, testCase.repoName)
// 		if result != testCase.expected {
// 			t.Errorf("%s FAILED, expected %s but got %s", logPrefix, testCase.expected, result)
// 		} else {
// 			t.Logf("%s PASSED, expected %s and got %s", logPrefix, testCase.expected, result)
// 		}
// 	}
// }

// type textActionTest struct {
// 	action     config.Action
// 	repoName   string
// 	fileData   []byte
// 	outputData []byte
// 	msg        string
// 	err        error
// }

// var replaceLineTests = []textActionTest{
// 	{
// 		config.Action{
// 			Type:   "replaceLine",
// 			Source: "NODE_ENV=test",
// 			Target: "NODE_ENV=development",
// 			Path:   ".env.example",
// 		},
// 		"node-boilerplate",
// 		[]byte("# Sets env var\nNODE_ENV=development\nHTTP_PORT=8080\n"),
// 		[]byte("# Sets env var\nNODE_ENV=test\nHTTP_PORT=8080\n"),
// 		"Replaced line `NODE_ENV=development` with `NODE_ENV=test` in `.env.example`",
// 		nil,
// 	},
// 	{
// 		config.Action{
// 			Type:   "replaceLine",
// 			Source: "NODE_ENV=test",
// 			Target: "NODE_ENV=development",
// 			Path:   ".env.example",
// 		},
// 		"node-boilerplate",
// 		[]byte("# Sets env var\nNODE_ENV=development\nHTTP_PORT=8080\n"),
// 		[]byte("# Sets env var\nNODE_ENV=test\nHTTP_PORT=8080\n"),
// 		"Replaced line `NODE_ENV=development` with `NODE_ENV=test` in `.env.example`",
// 		nil,
// 	},
// }

// func TestReplaceLine(t *testing.T) {
// 	for _, testCase := range replaceLineTests {
// 		outputData, msg, err := ReplaceLine(testCase.action, testCase.repoName, testCase.fileData)

// 		logPrefix := fmt.Sprintf("ReplaceLine(%+v, %s):", testCase.action, testCase.repoName)
// 		if !bytes.Equal(outputData, testCase.outputData) {
// 			t.Errorf("%s FAILED, outputData bytes are not equal", logPrefix)
// 		} else if msg != testCase.msg {
// 			t.Errorf("%s FAILED, expected msg '%s' but got '%s'", logPrefix, testCase.msg, msg)
// 		} else if err != testCase.err {
// 			t.Errorf("%s FAILED, expected err %s but got %s", logPrefix, err, testCase.err)
// 		} else {
// 			t.Logf("%s PASSED, all return values equal, %v, %s, %v", logPrefix, outputData, msg, err)
// 		}
// 	}
// }

// func TestDeleteLineError(t *testing.T) {

// }
