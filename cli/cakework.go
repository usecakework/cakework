package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

//go:embed Dockerfile
var dockerfile embed.FS

//go:embed fly.toml
var flyConfig embed.FS

// TODO put stuff into templates for different languages

//go:embed .gitignore_python
var gitIgnore embed.FS // for python only! TODO fix

var configFile string
var config Config

type Config struct {
	UserId       string `json:"userId"`
	App          string `json:"app"` // don't actually need this
	AccessToken  string
	RefreshToken string
}

func main() {
	var appName string
	var language string
	var appDirectory string

	workingDirectory, _ := os.Getwd()
	// buildDirectory := filepath.Join(workingDirectory, "build") // TODO figure out how to obfuscate all build files
	buildDirectory := workingDirectory
	dirname, _ := os.UserHomeDir()
	fly := dirname + "/.cakework/.fly/bin/fly" // TODO join paths in an OS specific way with proper delimiters
	cakeworkDirectory := dirname + "/.cakework"

	// initialize the config object
	configFile = filepath.Join(cakeworkDirectory, "config.json")
	var jsonFile *os.File
	if _, err := os.Stat(configFile); err == nil {
		jsonFile, err = os.Open(configFile)
		check(err)
	} else {
		jsonFile, err = os.Create(configFile)
		check(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	check(err)

	json.Unmarshal(byteValue, &config)

	// try to read appName from file, if it exists
	// log.SetLevel(log.InfoLevel)

	// TODO: don't dump out all outputs to cli for user to see
	// TODO provide access token option so user doesn't need to be authed via the cli (can deploy via gh actions for example)

	app := &cli.App{
		Name:     "cakework",
		Usage:    "This is the Cakework command line interface",
		Version:  "v1.0.65", // TODO figure out how to automatically update this
		Compiled: time.Now(),
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose", Hidden: true},
		},
		Authors: []*cli.Author{
			{
				Name:  "Jessie Young",
				Email: "jessie@cakework.com",
			},
		},
		Before: func(cCtx *cli.Context) error {
			if !cCtx.Bool("verbose") {
				log.SetLevel(log.ErrorLevel) // default behavior (not verbose) is to not log anything.
			} else {
				log.SetLevel(log.DebugLevel)
			}
			return nil
		},
		Commands: []*cli.Command{
			{ // if don't get the result within x seconds, kill
				Name:  "login", // TODO change this to signup. // TODO also create a logout
				Usage: "Authenticate the Cakework CLI",
				Action: func(cCtx *cli.Context) error {
					if isLoggedIn() {
						fmt.Println("You are already logged in 🍰")
						return nil
					}

					// when we auth (sign up or log in) for the first time, obtain a set of tokens
					err = auth()
					check(err)
					fmt.Println("You are logged in 🍰")
					return nil
				},
			},
			{
				Name:  "signup", // TODO change this to signup. // TODO also create a logout
				Usage: "Sign up for Cakework",
				Action: func(cCtx *cli.Context) error {
					if isLoggedIn() {
						fmt.Println("You are already logged in 🍰")
						return nil
					}
					err := auth()
					check(err)

					// create new user by calling Cakework frontend
					userId := getUserId()

					user := getUser(userId, config.AccessToken, config.RefreshToken)

					if user != nil {
						fmt.Println("You already have an account. Please log in instead.")
						return nil
					}

					user = createUser(userId)
					if user == nil {
						cli.Exit("Sign up failed", 1)
					}

					fmt.Println("Thanks for signing up with Cakework 🍰")

					return nil
				},
			},
			{
				Name:  "logout",
				Usage: "Log out of the Cakework CLI",
				Action: func(cCtx *cli.Context) error {
					err := os.Remove(configFile)
					if err != nil {
						cli.Exit("Failed to log out and delete Cakework config file", 1)
					}
					fmt.Println("You have been logged out")
					return nil
				},
			},
			{
				Name:      "create-client-token", // TODO change this to signup. // TODO also create a logout
				Usage:     "Create an access token for your clients",
				UsageText: "cakework create-client-token [TOKEN_NAME] [command options] [arguments...]",
				Action: func(cCtx *cli.Context) error {
					var name string
					if cCtx.NArg() > 0 {
						name = cCtx.Args().Get(0)
						// write out app name to config file
						// TODO in the future we won't
						// TODO write this out in json form

					} else {
						return cli.Exit("Please specify a name for the client token", 1)
					}

					if isLoggedIn() {
						userId := getUserId()
						log.Debug("got user id")
						clientToken := createClientToken(userId, name)

						fmt.Println("Created client token:")
						fmt.Println(clientToken.Token)
						fmt.Println("Store this token securely. You will not be able to see this again after initial creation.")

					} else {
						fmt.Println("Please sign up or log in first")
					}
					return nil
				},
			},
			{
				Name:      "new",
				Usage:     "Create a new project",
				UsageText: "cakework new [PROJECT_NAME] [command options] [arguments...]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "lang",
						Value:       "python",
						Usage:       "language for the project. Defaults to python 3.8",
						Destination: &language,
					},
				},
				Action: func(cCtx *cli.Context) error {
					if !isLoggedIn() {
						fmt.Println("Please sign up or log in first")
						return nil
					}

					if cCtx.NArg() > 0 {
						appName = cCtx.Args().Get(0)
						// write out app name to config file
						// TODO in the future we won't
						// TODO write this out in json form
						addConfigValue("App", appName)

					} else {
						return cli.Exit("Please include a Project name", 1)
					}
					lang := cCtx.String("lang")
					if lang == "python" {
						// Q: why isn't this getting printed out?
					} else {
						return cli.Exit("Language "+lang+" not supported", 1)
					}
					fmt.Println("Creating your new Cakework project " + appName + "...")

					s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
					s.Start()                                                   // Start the spinner

					err := os.Mkdir(appName, os.ModePerm)
					check(err)

					appDirectory = filepath.Join(buildDirectory, appName)

					text, err := gitIgnore.ReadFile(".gitignore_python")
					check(err)

					os.WriteFile(filepath.Join(appDirectory, ".gitignore"), text, 0644)

					srcDirectory := filepath.Join(appDirectory, "src")

					err = os.Mkdir(srcDirectory, os.ModePerm)
					check(err)

					main := `from cakework import Cakework
import time

def say_hello(name):
    time.sleep(5)
    return "Hello " + name + "!"

if __name__ == "__main__":
    cakework = Cakework("` + appName + `")
    cakework.add_task(say_hello)
`

					f, err := os.Create(filepath.Join(srcDirectory, "main.py"))
					check(err)
					defer f.Close()

					f.WriteString(main)
					f.Sync()

					// copy Dockerfile into current build directory
					text, err = dockerfile.ReadFile("Dockerfile")
					check(err)
					os.WriteFile(filepath.Join(appDirectory, "Dockerfile"), text, 0644)

					// TODO debug why this isn't working. for now we have a workaround
					f, err = os.Create(filepath.Join(appDirectory, ".dockerignore"))
					check(err)
					defer f.Close()

					f.WriteString("env")
					f.Sync()

					// TODO check python version
					cmd := exec.Command("python3", "-m", "venv", "env")
					cmd.Dir = appDirectory
					_, err = shell(cmd) // don't do anything with out?
					check(err)

					cmd = exec.Command("bash", "-c", "source env/bin/activate && pip3 install --upgrade setuptools pip && pip3 install --force-reinstall cakework")
					cmd.Dir = appDirectory
					_, err = shell(cmd) // don't do anything with out?
					check(err)

					cmd = exec.Command("bash", "-c", "source env/bin/activate; pip3 freeze")
					cmd.Dir = appDirectory

					// open the out file for writing
					outfile, err := os.Create(filepath.Join(appDirectory, "requirements.txt"))
					check(err)

					defer outfile.Close()
					cmd.Stdout = outfile

					err = cmd.Start()
					check(err)
					cmd.Wait()

					createExampleClient(appDirectory, appName)

					s.Stop()

					// TODO: will say done even when error out. need to fix!
					fmt.Println("Done creating your new project! 🍰")
					return nil
				},
			},
			{
				Name:  "deploy",
				Usage: "Deploy your Project",
				Action: func(cCtx *cli.Context) error {
					// TODO need to check if we are logged in before deploying!!
					// TODO: how to set the verbosity for every app?
					// TODO: should we only allow allowd users to call this action? so as long as someone has the user id in the file then it's ok?
					if !isLoggedIn() {
						return cli.Exit("Please sign up or log in first", 1)
					}

					fmt.Println("Deploying Your Project...")
					// find the app name
					readFile, err := os.Open(filepath.Join(filepath.Join(workingDirectory, "src"), "main.py"))

					// TODO add proper error handling
					if err != nil {
						log.Debug(err)
					}
					fileScanner := bufio.NewScanner(readFile)

					fileScanner.Split(bufio.ScanLines)

					var rgxAppName = regexp.MustCompile(`\(\"([^)]+)\"\)`)

					var appName string

					// TODO this is janky. can now get app name from config; how to make this less janky for getting the registered activity name?
					for fileScanner.Scan() {
						line := fileScanner.Text()
						if strings.Contains(line, "Cakework(") {
							rs := rgxAppName.FindAllStringSubmatch(line, -1)
							for _, i := range rs {
								appName = i[1]
							}
						}
					}

					if appName == "" {
						return cli.Exit("Failed to parse project name from main.py. Please make sure you're in the project directory!", 1)
					}
					readFile.Close()

					// sanitize activity name and app name. in the future we don't need to do this anymore
					appName = strings.ReplaceAll(strings.ToLower(appName), "_", "-") // in the future, infer these from the code

					// TODO do input validation for not allowed characters
					// userId := strings.ReplaceAll(strings.ToLower(cCtx.Args().First()), "_", "-") // in the future, infer these from the code

					// parse main.py to get the app name and task name
					// TODO: fix it so that we're not parsing python code from here
					readFile, err = os.Open(filepath.Join(filepath.Join(workingDirectory, "src"), "main.py"))

					// TODO add proper error handling
					if err != nil {
						log.Debug(err)
					}
					fileScanner = bufio.NewScanner(readFile)

					fileScanner.Split(bufio.ScanLines)

					var rgxTaskName = regexp.MustCompile(`\(([^)]+)\)`)

					var taskName string

					// TODO this is janky. can now get app name from config; how to make this less janky for getting the registered activity name?
					for fileScanner.Scan() {
						line := fileScanner.Text()
						if strings.Contains(line, "add_task") {
							rs := rgxTaskName.FindAllStringSubmatch(line, -1)
							for _, i := range rs {
								taskName = i[1]
							}
						}
					}

					if taskName == "" {
						return cli.Exit("Failed to parse task name from main.py. Please make sure you're in the project directory!", 1)
					}
					readFile.Close()

					// TODO: do we even need to store the app name in the config file?

					// sanitize activity name and app name. in the future we don't need to do this anymore
					appName = strings.ReplaceAll(strings.ToLower(appName), "_", "-") // in the future, infer these from the code

					addConfigValue("App", appName)                                     // TODO change to project
					taskName = strings.ReplaceAll(strings.ToLower(taskName), "_", "-") // in the future, infer these from the code

					userId := getUserId()

					flyAppName := userId + "-" + appName + "-" + taskName

					s := spinner.New(spinner.CharSets[9], 100*time.Millisecond) // Build our new spinner
					s.Start()                                                   // Start the spinner

					// TODO: instead of just deleting and re-creating the app, we can just delete the old machines
					// TODO figure out how to deploy fly machine instead of fly app
					// if name is already taken, want to make sure we don't overwrite it; need to make use of the old fly.toml. Should we store that for the user?

					// copy fly.toml
					text, _ := flyConfig.ReadFile("fly.toml")
					os.WriteFile(filepath.Join(buildDirectory, "fly.toml"), text, 0644)

					// update the fly.toml file
					flyConfig := filepath.Join(buildDirectory, "fly.toml")
					input, err := os.ReadFile(flyConfig)
					check(err)

					lines := strings.Split(string(input), "\n")

					// note: this is brittle (what if they don't have app with space?)
					for i, line := range lines {
						if strings.Contains(line, "app =") {
							lines[i] = "app = \"" + flyAppName + "\""
						}
					}
					output := strings.Join(lines, "\n")
					err = ioutil.WriteFile(flyConfig, []byte(output), 0644)
					check(err)

					// TODO remove access token from source code and re-create github repo
					cmd := exec.Command(fly, "apps", "create", flyAppName, "--org", "sahale", "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")
					cmd.Dir = buildDirectory

					var outCreate bytes.Buffer
					var stderrCreate bytes.Buffer
					cmd.Stdout = &outCreate
					cmd.Stderr = &stderrCreate
					errCreate := cmd.Run()
					allOutput := fmt.Sprint(err) + ": " + stderrCreate.String() + ": " + outCreate.String()

					if errCreate != nil {
						if strings.Contains(allOutput, "Name has already been taken") {
							log.Debug("fly app already exists. Not an error") // TODO in the future, we don't want to overwrite different people's code. will be fixed once we have auth

							log.Debug("allocating ip addresses") // q: need both ipv4 and ipv6? need ipv4 for sure.
							// TODO if they've been allocated successfully already, no need to re-do
							cmd = exec.Command(fly, "ips", "allocate-v4", "--app", flyAppName, "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")
							cmd.Dir = buildDirectory
							shell(cmd)
							// cmd = exec.Command(fly, "ips", "allocate-v6", "--app", flyAppName, "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")
							// cmd.Dir = buildDirectory
							// shell(cmd)

							cmd = exec.Command(fly, "deploy", "--app", flyAppName, "--region", "sea", "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")
							cmd.Dir = buildDirectory
							_, err := shell(cmd)
							check(err)

						} else {
							log.Debug(outCreate) // should we output more stuff here? like
							check(err)
						}
					} else { // deploying for the first time

						log.Debug("allocating ip addresses") // q: need both ipv4 and ipv6? need ipv4 for sure.
						// TODO if they've been allocated successfully already, no need to re-do
						cmd = exec.Command(fly, "ips", "allocate-v4", "--app", flyAppName, "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")
						cmd.Dir = buildDirectory
						shell(cmd)
						// cmd = exec.Command(fly, "ips", "allocate-v6", "--app", flyAppName, "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")
						// cmd.Dir = buildDirectory
						// shell(cmd)

						cmd = exec.Command(fly, "deploy", "--app", flyAppName, "--region", "sea", "--access-token", "QCMUb_9WFgHAZkjd3lb6b1BjVV3eDtmBkeEgYF8Mrzo")

						cmd.Dir = buildDirectory
						_, err := shell(cmd)
						check(err)

					}

					// creating a brand new app
					// TODO if there are old machines with old code, destroy them all first
					// out, err = shell(exec.Command("cd", directory + "/build/" + activityName)) // does this cd command actually do anything? since we set the dir
					// TODO switch back to using machine

					// TODO: if ips already allocated, no need to allocate more
					// maybe instead of fly launch, just do a deploy as well
					s.Stop()

					// delete fly.toml file
					// TODO make sure this block always runs even if everything else fails (put in finally block)
					e := os.Remove(filepath.Join(buildDirectory, "fly.toml"))
					check(e)

					fmt.Println("Successfully deployed your tasks! 🍰")

					//TODO if there is an error, then return nil

					// log.Debug("Success! Your app is now live 🍰")

					return nil
				},
			}, {
				Name:  "task",
				Usage: "Interact with your Tasks (e.g. get logs)",
				Subcommands: []*cli.Command{
					{
						Name:      "logs",
						Usage:     "Get request logs for a task",
						UsageText: "cakework task status [PROJECT_NAME] [TASK_NAME] [command options]",
						// Flags: []cli.Flag{
						// 	&cli.StringFlag{
						// 		Name:        "status",
						// 		Value:       "",
						// 		Usage:       "Status to filter by. PENDING, IN_PROGRESS, SUCCEEDED, or FAILED",
						// 		Destination: &status,
						// 	},
						// },
						Action: func(cCtx *cli.Context) error {
							if cCtx.NArg() != 2 {
								return cli.Exit("Please specify 2 parameters - Project name and Task Name.", 1)
							}

							appName := cCtx.Args().Get(0)
							taskName := cCtx.Args().Get(1)

							var statuses []string
							// status := cCtx.String("status")
							// if status != "" {
							// 	statuses = append(statuses, status)
							// }

							userId := getUserId()
							taskLogs := getTaskLogs(userId, appName, taskName, statuses)

							if len(taskLogs.Requests) == 0 {
								fmt.Println("Task " + appName + "/" + taskName + " does not exist, or you haven't run the task yet.")
								return nil
							}

							t := table.NewWriter()
							t.SetOutputMirror(os.Stdout)
							t.AppendHeader(table.Row{"Request Id", "Status", "Parameters", "Result"})
							for _, request := range taskLogs.Requests {
								t.AppendRow([]interface{}{
									request.RequestId,
									request.Status,
									request.Parameters,
									request.Result,
								})
							}
							t.Render()

							return nil
						},
					},
				},
			}, {
				Name:  "request",
				Usage: "Interact with your Requests (e.g. get logs)",
				Subcommands: []*cli.Command{
					{
						Name:      "status",
						Usage:     "Get processing status for your single request",
						UsageText: "cakework request status [REQUEST_ID]",
						Action: func(cCtx *cli.Context) error {
							if cCtx.NArg() != 1 {
								return cli.Exit("Please include one parameter, the Request ID", 1)
							}

							userId := getUserId()
							requestId := cCtx.Args().Get(0)

							requestStatus := getRequestStatus(userId, requestId)

							if requestStatus != "" {
								fmt.Println(requestStatus)
							}

							return nil
						},
					},
				},
			},
		},
	}

	err = app.Run(os.Args)
	check(err)
}

func createExampleClient(appDirectory string, appName string) {
	// create sample client
	exampleClientDirectory := filepath.Join(appDirectory, "example_client")
	err := os.Mkdir(exampleClientDirectory, os.ModePerm)
	check(err)
	exampleClient := `from cakework import Client
import time

# Generate your client token with create-client-token my-client-token
CAKEWORK_CLIENT_TOKEN = "YOUR_CLIENT_TOKEN_HERE"

if __name__ == "__main__":
    client = Client("` + appName + `", CAKEWORK_CLIENT_TOKEN)

    # You can persist this request ID to get status of the job later
    request_id = client.say_hello("from Cakework")

    status = client.get_status(request_id)
    while (status == "PENDING" or status == "IN_PROGRESS"):
        print("Still baking...!")
        status = client.get_status(request_id)
        time.sleep(1)

    if (client.get_status(request_id) == "SUCCEEDED"):
        result = client.get_result(request_id)
        print(result)
`

	f, err := os.Create(filepath.Join(exampleClientDirectory, "main.py"))
	check(err)
	defer f.Close()

	f.WriteString(exampleClient)
	f.Sync()
}

func check(e error) cli.ExitCoder {
	if e != nil {
		fmt.Println(e)
		return cli.Exit("Failed", 1)
		// TODO how to cause the program to exit?
	}
	return nil
}

func Check(e error) cli.ExitCoder {
	if e != nil {
		fmt.Println(e)
		return cli.Exit("Failed", 1)
		// TODO how to cause the program to exit?
	}
	return nil
}

func CheckPanic(e error) cli.ExitCoder {
	if e != nil {
		fmt.Println(e)
		return cli.Exit("Failed", 1)
		// TODO how to cause the program to exit?
	}
	return nil
}

func shell(cmd *exec.Cmd) (string, error) {
	log.Debug("executing command: " + strings.Join(cmd.Args, " ")) // TODO turn this off when not in debug mode
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	allOutput := fmt.Sprint(err) + ": " + stderr.String() + ": " + out.String()

	if err != nil {
		log.Debug("error executing command")
		log.Debug("err: ")
		log.Debug(fmt.Sprint(err)) // TODO delete this
		log.Debug("stderr: (may be null): ")
		log.Debug(stderr.String())
		log.Debug("out: (may be null)")
		log.Debug(out.String())
		fmt.Println("out") // TODO remove these so that we obfuscate errors from the user
		fmt.Println(out.String())
		fmt.Println("err")
		fmt.Println(err)
		fmt.Println("stderr")
		fmt.Println(stderr.String())
		// since sometimes errors are printed to stdout instead of stderr, print out stdout as well
		return allOutput, err
	} else {
		log.Debug("succeeded executing command")
	}
	log.Debug("Result: (out)" + out.String())
	return out.String(), nil
}

// File copies a single file from src to dst
func File(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

// Dir copies a whole directory recursively
func Dir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = Dir(srcfp, dstfp); err != nil {
				log.Error(err)
			}
		} else {
			if err = File(srcfp, dstfp); err != nil {
				log.Error(err)
			}
		}
	}
	return nil
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	check(err)
}

func responseWithJSON(writer io.Writer, content []byte, status int) {
}

func errWithJSON(writeri io.Writer, content string, status int) {

}

func verifyTokenController(w http.ResponseWriter, r *http.Request) {
	prefix := "Bearer "
	authHeader := r.Header.Get("Authorization")
	reqToken := strings.TrimPrefix(authHeader, prefix)

	// log.Println(reqToken)

	if authHeader == "" || reqToken == authHeader {
		errWithJSON(w, "Authentication header not present or malformed", http.StatusUnauthorized)
		return
	}
	// log.Println(reqToken)
	responseWithJSON(w, []byte(`{"message":"Token is valid"}`), http.StatusOK)

}

// TODO possibly  move this to the auth.go
func auth() error {
	log.Debug("Starting log in flow")
	var data map[string]interface{}

	// if exists already and a user is found, can skip the log in

	// request device code
	// TODO: instead of logging into the cli, we should be logging into/getting credentials for the api?
	url := "https://dev-qanxtedlpguucmz5.us.auth0.com/oauth/device/code"

	// if using the creds to call an api, need to use the API's Identifier as the audience
	payload := strings.NewReader("client_id=rqbQ3XWpM2C0vRCzKwC6CXXnKe9aCSmb&scope=openid offline_access add:task get:user create:user create:client_token get:status get:task_status %7D&audience=https%3A%2F%2Fcakework-frontend.fly.dev")

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	bodyString, data, res := callHttp(req)

	if res.StatusCode != 200 {
		log.Debug(res.StatusCode)
		log.Debug(res)
		log.Debug(bodyString)
		return cli.Exit("Failed to log in using device code", 1)
	}

	verificationUrl := data["verification_uri_complete"].(string)

	deviceCode := data["device_code"].(string)
	userCode := data["user_code"].(string)
	fmt.Println("User code: " + userCode)

	openBrowser(verificationUrl)

	var accessToken string
	var refreshToken string
	// poll for request token
	// Q: make it so that we only try for up to X minutes
	for {
		url = "https://dev-qanxtedlpguucmz5.us.auth0.com/oauth/token"

		payload = strings.NewReader("grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=" + deviceCode + "&client_id=rqbQ3XWpM2C0vRCzKwC6CXXnKe9aCSmb")

		req, _ = http.NewRequest("POST", url, payload)

		log.Debug("payload to /token endpoint:")
		req.Header.Add("content-type", "application/x-www-form-urlencoded")
		// req.Header.Add("content-type", "application/json")

		res, _ = http.DefaultClient.Do(req)

		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)

		stringBody := string(body)

		var data map[string]interface{}
		err := json.Unmarshal([]byte(string(body)), &data)
		check(err)

		if strings.Contains(stringBody, "access_token") {
			log.Debug("Successfully got an access token!")

			accessToken = data["access_token"].(string)
			refreshToken = data["refresh_token"].(string)

			break
		} else {
			time.Sleep(5 * time.Second) // TODO actually get the interval from above
		}
	}

	// keep this?
	if accessToken == "" {
		fmt.Println("Failed to fetch an access token") // is there error handling to do here?
		return nil
	}
	if refreshToken == "Failed to fetch a refresh token" {
		fmt.Println("Failed to fetch an refresh token")
		return nil
	}

	log.Debug("access_token: " + accessToken)
	log.Debug("refresh_token: " + refreshToken)

	addConfigValue("AccessToken", accessToken)
	addConfigValue("RefreshToken", refreshToken)

	// TODO: we should store the accessToken and refreshToken
	// call the /userInfo API to get the user information

	// technically don't need to make a call to this; can parse the jwt token to get the sub field.
	url = "https://dev-qanxtedlpguucmz5.us.auth0.com/userinfo"

	req, _ = newRequestWithAuth("GET", url, nil)
	stringBody, data, res := callHttp(req)

	if res.StatusCode != 200 {
		log.Debug(res.StatusCode)
		log.Debug(res)
		log.Debug(stringBody)
		return cli.Exit("Failed to get user info", 1)
	}

	sub := data["sub"].(string)
	userId := strings.Split(sub, "|")[1]
	log.Debug("Got userId: " + userId) // TODO delete this
	addConfigValue("UserId", userId)

	return nil
}

func isLoggedIn() bool {
	if _, err := os.Stat(configFile); err == nil {
		jsonFile, err := os.Open(configFile)
		check(err)
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()

		var config Config

		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &config)
		if config.UserId != "" {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func loadConfig() Config {
	var config Config
	if _, err := os.Stat(configFile); err == nil {
		jsonFile, err := os.Open(configFile)
		check(err)
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()

		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &config)
	}
	return config
}

// should only call if a user is logged in
func getUserId() string {
	jsonFile, err := os.Open(configFile)
	check(err)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	var config Config

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &config)
	return config.UserId // TODO may want to do some checks. assume this returns not ""
}

// this also writes to the config file
// note: field name needs to be in all caps!
func addConfigValue(field string, value string) {
	v := reflect.ValueOf(&config).Elem().FieldByName(field)
	if v.IsValid() {
		v.SetString(value)
	}

	file, _ := json.MarshalIndent(config, "", " ")

	err := ioutil.WriteFile(configFile, file, 0644)
	check(err)
}

// this also writes to the config file
func deleteConfigValue(field string) {
	v := reflect.ValueOf(&config).Elem().FieldByName("UserId")
	if v.IsValid() {
		v.SetString("")
	} else {
		// TODO should throw an exception
	}

	file, _ := json.MarshalIndent(config, "", " ")

	err := ioutil.WriteFile(configFile, file, 0644)
	check(err)
}

type CustomClaimsExample struct {
	Scope string `json:"scope"`
}

// Validate errors out if `ShouldReject` is true.
func (c *CustomClaimsExample) Validate(ctx context.Context) error {
	// if c.ShouldReject {
	// 	return errors.New("should reject was set to true")
	// }
	return nil
}

func (c *CustomClaimsExample) Valid() error {
	return nil
}

func refreshAndSaveTokens() {
	// fetch new tokens
	newToken, newRefreshToken := refreshTokens(config.AccessToken, config.RefreshToken)
	if newToken == "" || newRefreshToken == "" {
		fmt.Println("Failed to refresh token")
		os.Exit(1)
	} else {
		addConfigValue("AccessToken", newToken)
		addConfigValue("RefreshToken", newRefreshToken)
	}
}
