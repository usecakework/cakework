use lambda::error::{CreateFunctionError, UpdateFunctionCodeError};
use lambda::output::UpdateFunctionCodeOutput;
use log::{info, warn, error, debug};
use aws_config::meta::region::RegionProviderChain;
use aws_sdk_sqs::{Client, Region};
#[macro_use]
extern crate dotenv_codegen;
use anyhow::anyhow;
use std::env::{self, args};
mod error;
mod sqs_queue;
mod queue;
use std::path::Path;
use std::{time::Duration};
use apigateway::model::IntegrationType;
use futures::{stream, StreamExt};

pub use error::Error;
use lambda::{model::FunctionCode, types::Blob};
use sqs_queue::SqsQueue;
use queue::{Job, Queue};
use aws_sdk_sqs as sqs;
use aws_sdk_lambda as lambda;
use aws_sdk_apigatewayv2 as apigateway;

use zip::{self, write::FileOptions};

use std::io::prelude::*;

use sqlx::{Pool, MySql};
use sqlx::mysql::MySqlPoolOptions;

use std::process::Command; // for running shell commands
use std::io::{self, Write};


const CONCURRENCY: i32 = 10;
const API_GATEWAY_API_ID: &str = "6tgrfokbok"; // for beta stage

// how to have var args? 
// fn run_cmd(program: &str, args: [&str]) {
    
// }


// fn run_cmd(mut cmd: Command) { 
//     assert!(
//         cmd
//             .output()
//             .expect("command failed to start")
//             .status
//             .success()
//         ); 
// }

// fn run_cmd_args(cmd: &Command) { 
//     assert!(
//         cmd
//             .output()
//             .expect("command failed to start")
//             .status
//             .success()
//         ); 
// }

fn main() -> Result<(), anyhow::Error> {
    // TODO: see if pip exists on render. if not, install it
    // TODO make sure deactivate works. try running diff jobs in a row
    // get the job id
    env_logger::init();

    // TODO test this working with multiple workers 
    // TODO: just take the job_id
    let job_id = "env3"; // TODO generate a uuid

    let tmp_dir = Path::new("/tmp");
    let mut job_dir = tmp_dir.join(job_id);

    let output = Command::new("pip3")
        .arg("install")
        .arg("virtualenv")
            .output()
            .expect("pipe3 install virtualenv failed");

    println!("status: {}", output.status);
    println!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    println!("stderr: {}", String::from_utf8_lossy(&output.stderr));

    assert!(
        Command::new("virtualenv")
        .arg("-p")
        .arg("python3")
        .arg(&job_dir)
            .output()
            .expect("virtualenv failed")
            .status
            .success()
        );
 
    // TODO: uncomment this back!!!!!!!!!!
    // Command::new("bash")
    //     .arg("-c")
    //     .arg(format!("source {}", job_path.join("bin").join("activate").to_str().unwrap()))
    //     .output()
    //     .expect("Failed to source activate file");
    
    // TODO get list of packages and store in dependencies variable
    let dependencies = ["requests", "numpy"];
    // TODO: pipe the dependencies into a requirements.txt file
    // let dependencies = [];
    let mut pip_cmds = "".to_string();
    for dependency in dependencies {
        pip_cmds.push_str(" && pip3 install --target ./package ");
        pip_cmds.push_str(dependency);
    }
    
    let mut full_cmd = "source ".to_string();
    full_cmd.push_str(job_dir.join("bin").join("activate").to_str().unwrap());
    if ! dependencies.is_empty() {
        full_cmd.push_str(pip_cmds.as_str());
    }

    // if ! dependencies.is_empty() {
    //     full_cmd.push_str("pip3 install -r requirements.txt");
    // }

    full_cmd.push_str(" && deactivate");

    info!("full cmd: {:?}", full_cmd);

    // Q: if this fails how will we know? will panic? how to prevent panicking? do we recover after panicking?
    // Q: how to pip install multiple files? is calling pip multiple times slower? 
    let output = Command::new("bash")
        .arg("-c")
        .arg(full_cmd)
        .output()
        .expect("Failed to pip install");

    println!("status: {}", output.status);
    println!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    println!("stderr: {}", String::from_utf8_lossy(&output.stderr));
    

    let mut cd_package_cmd = "cd ".to_string();
    cd_package_cmd.push_str(job_dir.join("package").to_str().unwrap());
    cd_package_cmd.push_str(" && zip -r ../");
    cd_package_cmd.push_str(job_id);
    cd_package_cmd.push_str(".zip .");

    info!("cd package cmd: {:?}", cd_package_cmd.to_string());
    
    let output = Command::new("bash")
        .arg("-c")
        .arg(cd_package_cmd)
        .output()
        .expect("Failed to zip packages");

    println!("status: {}", output.status);
    println!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    println!("stderr: {}", String::from_utf8_lossy(&output.stderr));
    
    // zip together the dependencies package and the lambda function file
    // TODO save the file in the /tmp/job_id directory
    let mut zip_filename = job_id.to_string();
    zip_filename.push_str(".zip");
    let mut zip_together_cmd = "zip -g ".to_string();
    let mut zip_path = job_dir.join(&zip_filename.to_string());
    zip_together_cmd.push_str(zip_path.to_str().unwrap());
    zip_together_cmd.push_str(" /tmp/lambda_function.py");

    let output = Command::new("bash")
        .arg("-c")
        .arg(zip_together_cmd)
        .output()
        .expect("Failed to add lambda function to zip");

    println!("status: {}", output.status);
    println!("stdout: {}", String::from_utf8_lossy(&output.stdout));
    println!("stderr: {}", String::from_utf8_lossy(&output.stderr));


    // uncomment this
    // match std::fs::remove_dir_all(&job_dir) {
    //     Ok(_) => (),
    //     Err(e) => error!("failed to delete directory {:?}, {}", &job_dir, e)
    // };
    Ok(())
}

// // Intermediate struct for reading from the MySQL db using using bytes as Strings, due to sqlx bug
// // need to manually convert into an Action
// #[derive(sqlx::FromRow)]
// #[derive(Debug)]
// struct ActionVec {
//     id: Vec<u8>,
//     createdAt: chrono::DateTime<chrono::Utc>,
//     updatedAt: chrono::DateTime<chrono::Utc>,
//     name: Vec<u8>, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     r#type: Vec<u8>, // can't just call it type since type is a reserved word in rust
//     code: Option<Vec<u8>>, // Q: are there different types of strings? fixed length, medium text, etc? This can be optional? 
//     appId: Vec<u8>,
//     restVerb: Option<Vec<u8>>, // optional? 
//     description: Option<Vec<u8>>, //optional? 
//     path: Option<Vec<u8>> // optional? 
// }

// #[derive(Debug)]
// struct Action {
//     id: String,
//     createdAt: chrono::DateTime<chrono::Utc>,
//     updatedAt: chrono::DateTime<chrono::Utc>,
//     name: String, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     r#type: String, // can't just call it type since type is a reserved word in rust
//     code: Option<String>, // Q: are there different types of strings? fixed length, medium text, etc? This can be optional? 
//     appId: String, // this shouldn't be optional?!
//     restVerb: Option<String>, // optional? 
//     description: Option<String>, //optional? 
//     path: Option<String> // optional? 
// }

// #[derive(sqlx::FromRow)]
// #[derive(Debug)]
// struct AppVec {
//     id: Vec<u8>,
//     createdAt: chrono::DateTime<chrono::Utc>,
//     updatedAt: chrono::DateTime<chrono::Utc>,
//     name: Vec<u8>, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     userId: Vec<u8> // can't just call it type since type is a reserved word in rust
// }

// #[derive(Debug)]
// struct App {
//     id: String,
//     createdAt: chrono::DateTime<chrono::Utc>,
//     updatedAt: chrono::DateTime<chrono::Utc>,
//     name: String, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     userId: String // can't just call it type since type is a reserved word in rust
// }

// #[derive(sqlx::FromRow)]
// #[derive(Debug)]
// struct UserVec {
//     id: Vec<u8>,
//     // createdAt: chrono::DateTime<chrono::Utc>,
//     // updatedAt: chrono::DateTime<chrono::Utc>,
//     // email: Vec<u8>, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     // familyName: Option<Vec<u8>>,
//     // givenName: Vec<u8>, // can't just call it type since type is a reserved word in rust
//     // identifier_token: Vec<u8>, // why is this using snake case instead of camel case?
//     // hashed_password: Option<Vec<u8>> // Q: why is this using snake case instead of camel case? 
// }

// #[derive(Debug)]
// struct User {
//     id: String,
//     // createdAt: chrono::DateTime<chrono::Utc>,
//     // updatedAt: chrono::DateTime<chrono::Utc>,
//     // email: String, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     // familyName: Option<String>,
//     // givenName: String, // can't just call it type since type is a reserved word in rust
//     // identifier_token: String, // why is this using snake case instead of camel case?
//     // hashed_password: Option<String> // Q: why is this using snake case instead of camel case? 
// }

// #[derive(sqlx::FromRow)]
// #[derive(Debug)]
// struct DeploymentVec {
//     id: Vec<u8>,
//     createdAt: chrono::DateTime<chrono::Utc>,
//     updatedAt: chrono::DateTime<chrono::Utc>,
//     actionId: Vec<u8>, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     status: Vec<u8>, // can't just call it type since type is a reserved word in rust
// }

// #[derive(Debug)]
// struct Deployment {
//     id: String,
//     createdAt: chrono::DateTime<chrono::Utc>,
//     updatedAt: chrono::DateTime<chrono::Utc>,
//     actionId: String, // mapped to Vec<u8> instead of String due to sqlx bug: https://github.com/launchbadge/sqlx/issues/1690 and https://github.com/planetscale/discussion/discussions/162
//     status: String, // can't just call it type since type is a reserved word in rust
// }

// const LAMBDA_HANDLER_CODE: &str = r#"
// import json

// def lambda_handler(event, context):
//     result = handle(None, None, None, None, None)
//     return {
//         'statusCode': 200,
//         'body': json.dumps(result)
//     }
// "#;


// #[tokio::main]
// async fn main() -> Result<(), anyhow::Error> {
//     env_logger::init();

//     run_cmd(Command::new("cd").arg("/tmp"));

//     let tmp_dir = "env1"; // TODO generate a uuid
//     run_cmd(
//         Command::new("virtualenv")
//         .arg("-p")
//         .arg("python3")
//         .arg(tmp_dir)
//     );
//     run_cmd(Command::new("cd").arg(tmp_dir));
//     run_cmd(&Command::new("bin/activate"));
//     info!("current working directory is:"); // TODO can remove this
//     run_cmd(&Command::new("pwd"));

//     run_cmd(&Command::new("bin/deactivate"));
//     // TODO delete the tmp directory
//     let mut dir: String = "../".to_string();    
//     dir.push_str(tmp_dir);
//     info!("directory is: {}", dir);

//     match std::fs::remove_dir_all(Path::new(&dir)) {
//         Ok(_) => (),
//         Err(e) => error!("failed to delete directory {}, {}", &dir, e)
//     };


//     panic!("no disco");
//     // create a new virtualenv
//     // Q: will block the current worker thread? 
//     //
    
    


//         // Command::new("pip3")
//         // .arg("install")
//         // .arg("numpy")
//         // .output()
//         // .expect("pip3 install command failed to start");

      

//     let database_url = dotenv!("DATABASE_URL");
//     let queue_name = dotenv!("SQS_QUEUE_NAME");
//     info!("Found DATABASE_URL: {} and queue name: {}", database_url, queue_name);

//     let pool = MySqlPoolOptions::new()
//         .max_connections(10) // TODO tune this
//         .connect(database_url).await?;

//     // TODO delete. For inserting a test record into the queue. 

//     // let job = SqsJob {
//     //     id: "qw2NsTR3JAgmfE".to_string(),
//     //     action_id: "qw2NsTR3JAgmfE".to_string(),
//     //     created_at: chrono::prelude::Utc::now(),
//     //     receipt_handle: "blah".to_string()
//     // };

//     // handle_job(job, pool.clone());

//     // let shared_config = aws_config::from_env().region(region_provider).load().await;
//     let shared_config = aws_config::load_from_env().await;

//     let lambda = lambda::Client::new(&shared_config);
//     let apig = apigateway::Client::new(&shared_config);
//     let sqs = sqs::Client::new(&shared_config);

//     let queue = SqsQueue::new(sqs, queue_name.to_string()).await;
//     run_poller(queue, pool, &lambda, &apig).await;
//     Ok(())
// }

// async fn run_poller(queue: SqsQueue, pool: Pool<sqlx::MySql>, lambda: &lambda::Client, apig: &apigateway::Client) {
//     loop {
//         let jobs = match queue.pull(CONCURRENCY).await {
//             Ok(jobs) => jobs,
//             Err(err) => {
//                 error!("error pulling from queue: {}", err);
//                 Vec::new()
//             }
//         };
 
//         let number_of_jobs = jobs.len();
//         if number_of_jobs > 0 {
//             info!("Fetched {} jobs", number_of_jobs);
//         }

//         stream::iter(jobs)
//             .for_each_concurrent(CONCURRENCY as usize, |job| async {
//                 // let receipt_handle = job.clone().receipt_handle; // TODO figure out better way instead of cloning. need this, otherwise get ownership error
//                 let res = match handle_job(&job, pool.clone(), lambda, apig).await { //Q: should we be cloning the pool? according to https://github.com/launchbadge/sqlx/discussions/917 this is kosher 🤷‍♀️
//                     Ok(_) => queue.delete_job(&job.receipt_handle.unwrap().as_str()).await,
//                     Err(e) => {
//                         error!("{:?}", e);
//                         Result::Err(Error::Internal(e.to_string())) // TODO: should throw/handle the error! For now just log it.
//                     }
//                 };
//             })
//             .await;

//         // sleep. may want to tweak this number
//         tokio::time::sleep(Duration::from_millis(250)).await;
//     }
// }

// // TODO: error handling. what happens if this throws an error? how do we do retries?
// async fn handle_job(job: &Job, pool: Pool<MySql>, lambda: &lambda::Client, apig: &apigateway::Client) -> Result<(), anyhow::Error> {// Result<(), anyhow::Error> { //Result<(), crate::Error> {
//     info!("Handling job: {:?}", &job.deployment_id);

//     let latest_deployment = get_latest_deployment(&job.action_id, pool.clone()).await?; // this should bubble up the error?
    
//     if latest_deployment.createdAt > job.created_at { // there's a diffrent, newer deployment in the db, abort
//         info!("newer deployment; skipping. new: {}, curent, {} ", latest_deployment.createdAt, job.created_at); // TODO fix the casing here
//         update_deployment_status(job.deployment_id.as_str(), "SKIPPED", pool.clone()).await?;

//         return Ok(());
//     }
    
//     // get jobs for the action from db; if there are any more recently created jobs, stop processing this current one

//     // TODO: check the db to make sure there's nothing else that's currently processing which is newer
//     // if there's a job which is newer, delete the current one
//     let action = get_action(job.action_id.as_str(), pool.clone()).await?;

//     let mut filename: String = job.deployment_id.to_string();
//     let extension = ".zip";
    
//     filename.push_str(extension);

//     let rest_verb = &action.restVerb.unwrap();
    
//     let path = action.path.unwrap();
//     let mut route_key: String = rest_verb.to_string();    
//     route_key.push_str(" /");
//     route_key.push_str(&path);

//     let lambda_name = path.replace("/", "_");

//     let mut function_code: String = LAMBDA_HANDLER_CODE.to_string();
//     function_code.push_str("\n");
//     function_code.push_str(&action.code.unwrap());

//     let zip = zip(&filename, &function_code.as_str());
//     let buffer = std::fs::read(&filename).unwrap();

//     let mut should_update_lambda: bool = false;
    
//     // TODO delete the zip files

//     // hack: need to not clone
//     let buffer2 = buffer.clone();

//     // let mut lambda_arn: &str = "dummy arn";

//     // rsp should contain the lambda arn
//     let lambda_arn: Result<String, error::Error> = match lambda.create_function()
//         .role("arn:aws:iam::605446105000:role/SahaleLambdaExecutionRole".to_string())
//         .code(FunctionCode::builder().set_zip_file(Some(Blob::new(buffer))).build()) // TODO add real code here
//         .function_name(&lambda_name)
//         .role("arn:aws:iam::605446105000:role/SahaleLambdaExecutionRole")
//         .runtime(lambda::model::Runtime::Python39)
//         .handler("lambda_function.lambda_handler".to_string())
//         .send()
//         .await {
//             Ok(rsp) => {
//                 info!("created function");
//                 Ok(rsp.function_arn().clone().unwrap().to_string())
//             },
//             Err(lambda::types::SdkError::ServiceError { err, .. }) => {
//                 if err.is_resource_conflict_exception() {
  
//                     info!("function already exists; try to update"); // TODO change to debug
//                     // should_update_lambda = true;

//                     match lambda
//                         .update_function_code()
//                         .function_name(&lambda_name)
//                         .zip_file(Blob::new(buffer2))
//                         .send()
//                         .await {
//                             Ok(rsp) => {
//                                 info!("successfully updated function code");
//                                 info!("got function arn from update call: {}", rsp.function_arn().clone().unwrap().to_string());
//                                 // lambda_arn = rsp.function_arn().to_owned().unwrap();
//                                 // Ok(rsp.function_arn().unwrap())
//                                 // Ok("blah")
//                                 Ok(rsp.function_arn().clone().unwrap().to_string())
//                                 },
//                             Err(e) => {
//                                 error!("error updating function code");
//                                 Err(error::Error::GeneralError(e.to_string()))? // does this show what error type? Q: does this bubble up? we want it to
//                             }
//                         }
            
//                 } else { // if it's a non resource conflict exception, it's a true exception; throw error  
//                     // Ok("sd")
//                     // Err(anyhow::anyhow!(err.to_string())) 
//                     error!("error creating function which wasn't a function already exists error");
//                     Err(error::Error::GeneralError(err.to_string()))? // does this bubble up? we want it to
//                 }
//             },
//             Err(e) => {
//                 error!("error with createFunction which wasn't a sdk error (not sure how this can even happen)");
//                 Err(error::Error::GeneralError(e.to_string()))?  // will this bubble up? what is the purpose of this here?
//             } 
//         };

//     // if should_update_lambda {
//     //     lambda_arn = match lambda
//     //     .update_function_code()
//     //     .function_name(&lambda_name)
//     //     .zip_file(Blob::new(buffer2))
//     //     .send()
//     //     .await {
//     //         Ok(rsp) => {
//     //             info!("successfully updated function code");
//     //             // lambda_arn = rsp.function_arn().to_owned().unwrap();
//     //             // Ok(rsp.function_arn().unwrap())
//     //              // Ok("blah")
//     //              Ok(Some(rsp.function_arn().clone().unwrap().to_string()))
//     //             },
//     //         Err(e) => {
//     //             Err(e)?
//     //         }
//     //      };
//     // }

//     // this should never happen since we bubble up the error?
//     if lambda_arn.is_err() {
//         error!("unable to fetch lambda arn. should have thrown error before this"); // TODO throw an error
        
//         match std::fs::remove_file(&filename) {
//             Ok(_) => (),
//             Err(e) => error!("failed to delete file {}, {}", &filename, e)
//         };
//         update_deployment_status(job.deployment_id.as_str(), "FAILED", pool.clone()).await?;
//         //TODO return error. for some reason it's not lettnig me do that, so return ok for now
//         return Ok(())
//     }

//     let lambda_arn = lambda_arn.unwrap(); // ok if this panics? 

//     // if lambda_arn.unwrap() != None { // is this the right check?
//     //     info!("got a lambda arn"); // TODO delete
//     // } else {
//     //     // TODO delete
//     //     info!("didn't get a lambda arn");
//     //     return Ok(())
//     // }

//     // let rsp = lambda.get_function()
//     //     .function_name(&lambda_name)
//     //     .send()
//     //     .await?;

//     // let lambda_arn = rsp.configuration().unwrap().function_arn().unwrap();
//     // println!("lambda arn: {:?} ", lambda_arn);

//     // TODO find a better way besides cloning each action
//     // TODO: make sure that whenever we have errors that we don't expose that we're using lambda under the hood

//     // pre-append the snippet with the format that lambda expects
//     // Q: how to get the other parameters such as headers in there? 
//     // TODO: don't create a new client for each request. figure out how to share resources. 
//     // TODO delete temp zip file

//     // if we ever need to update the route name, then would need to re-do this
//      let route_id: Result<String, error::Error> = match apig.create_route() // need to annotate even though never used
//         .api_id(API_GATEWAY_API_ID)
//         .route_key(&route_key) // TODO make the route name configurable based on the action name
//         .send()
//         .await {
//             Ok(rsp) => {
//                 info!("created route with key: {:?} ", &route_key);
//                 Ok(rsp.route_id.unwrap())
//             },
//             Err(apigateway::types::SdkError::ServiceError { err, .. }) => {
//                 if err.is_conflict_exception() {
//                     let mut _route_id: String;
  
//                     // this will fail if the following steps never succeeded
//                     info!("couldn't create route; already exists. finding route id"); // TODO change to debug
//                     let mut found_route_id = false;
//                     // TODO do this with pagination. assume no pagination for now

//                     let route_id = find_route_id(API_GATEWAY_API_ID, &route_key, apig).await?;
//                     Ok(route_id)
//                     //   let rsp = apig.get_routes()
//                 //             .api_id(≈)
//                 //             .set_next_token(next_token.clone())
//                 //             .send()
//                 //             .await?;
            
//                 //             let routes = rsp.items().unwrap();
//                 //             info!("found {} routes", routes.len());
            
//                 //              // TODO optimize this. right now taking too long cuz iterating through every single route? 
//                 //         for route in routes {
//                 //             if found_route_id == false {
            
//                 //                 if route.route_key().unwrap().eq(&route_key) {
//                 //                     found_route_id = true;
//                 //                     info!("found route id!");

//                 //                     _route_id = route.route_id().unwrap().to_string();

//                     // let mut next_token: Option<String> = None;


//                     // should only execute this if we didn't just create a route. 
//                 //     loop {
//                 //         // get the route id
//                 //         let rsp = apig.get_routes()
//                 //             .api_id(API_GATEWAY_API_ID)
//                 //             .set_next_token(next_token.clone())
//                 //             .send()
//                 //             .await?;
            
//                 //             let routes = rsp.items().unwrap();
//                 //             info!("found {} routes", routes.len());
            
//                 //              // TODO optimize this. right now taking too long cuz iterating through every single route? 
//                 //         for route in routes {
//                 //             if found_route_id == false {
            
//                 //                 if route.route_key().unwrap().eq(&route_key) {
//                 //                     found_route_id = true;
//                 //                     info!("found route id!");

//                 //                     _route_id = route.route_id().unwrap().to_string();
//                 //                     break;
                                    
//                 //             }
//                 //         } else {
//                 //             break;
//                 //         }
//                 //         if rsp.next_token().is_none() {
//                 //             break; 
//                 //         } 
//                 //         if found_route_id == true {
//                 //             break;
//                 //         }
//                 //         next_token = Some(rsp.next_token().unwrap().to_string());
//                 //     }
//                 // }
//                 // Ok(_route_id);
                
//                 } else { // if it's a non resource conflict exception, it's a true exception; throw error  
//                     // Ok("sd")
//                     // Err(anyhow::anyhow!(err.to_string())) 
//                     error!("error creating route which wasn't a route already exists error");
//                     Err(error::Error::GeneralError(err.to_string()))? // does this bubble up? we want it to
//                 }
//             },
//             Err(e) => {
//                 error!("error with create route which wasn't a sdk error (not sure how this can even happen)");
//                 Err(error::Error::GeneralError(e.to_string()))?  // will this bubble up? what is the purpose of this here?
//             } 
//         };

        
//     let route_id = route_id.unwrap(); // this may panic
//     info!("found route id: {}", route_id);


//     // now that we have a route id, create integrations
//     let mut integration_uri = "arn:aws:apigateway:us-west-2:lambda:path/2015-03-31/functions/".to_string();
//     integration_uri.push_str(&lambda_arn);
//     integration_uri.push_str("/invocations");
//     // println!("integration uri: {:?} ", integration_uri);
//     // check that integration with the lambda function already exists for the route
//     // TODO: if integration already exists, no need to create a new one
//     // get the route and get its target if one exists

//     let mut should_create_integration: bool = true;
//     let get_route_rsp = match apig.get_route()
//         .api_id(API_GATEWAY_API_ID)
//         .route_id(&route_id)
//         .send()
//         .await {
//             Ok(rsp) => {
//                 let target = match rsp.target() {
//                     Some(_) => {
//                         should_create_integration = false;
//                         info!("successfully got route. integration already exists; skip creating"); // TODO check that integration is for the correct lambda
//                     },
//                     None => {
//                         info!("successfully got route. integration doesn't exist; need to create");
//                     }
//                 };
//                 Ok(rsp)
//             },
//             Err(e) => {
//                 error!("failed to get route, {} ", e);
//                 Err(e)
//             }
//         };

//         // fix logic - sometimes integration wasn't created properly even though route was
    
//     if should_create_integration {
//         info!("creating integration");
//         let create_integration_rsp = match apig.create_integration()
//         .api_id(API_GATEWAY_API_ID)
//         .integration_type(IntegrationType::AwsProxy)
//         .integration_uri(integration_uri)
//         .payload_format_version("2.0")
//         .send()
//         .await {
//             Ok(rsp) => {
//                 info!("successfully created integration");
//                 Ok(rsp)
//             },
//             Err(e) => {
//                 error!("failed to create integration: {:?} ", e);
//                 Err(e)
//             }
//         }?;

//         let mut route_arn = "arn:aws:execute-api:us-west-2:605446105000:".to_string();
//         route_arn.push_str(API_GATEWAY_API_ID);
//         route_arn.push_str("/*/");
//         route_arn.push_str(rest_verb);
//         route_arn.push_str("/");
//         route_arn.push_str(&path);

//         info!("Route arn: {:?} ", route_arn);

//         // api-id/stage/http-method/resource-path"
//         // add permissions
//         // note: don't do this if already done
//         let add_permission_rsp = match lambda.add_permission()
//             .function_name(&lambda_name)
//             .statement_id("api-gateway-invoke")
//             .action("lambda:InvokeFunction")
//             .source_arn(route_arn)
//             .principal("apigateway.amazonaws.com")
//             .send()
//             .await {
//                 Ok(rsp) => {
//                     info!("successfully added permission");
//                     Ok(rsp)
//                 },
//                 Err(e) => {
//                     error!("failed to add permission with {:?} ", e);
//                     Err(e)
//                 }
//             };

//         let integration_id = create_integration_rsp.integration_id().unwrap();
//         let update_route_rsp = match apig.update_route()
//             .api_id(API_GATEWAY_API_ID)
//             .route_id(route_id)
//             .target(["integrations/", integration_id].join(""))
//             .send()
//             .await {
//                 Ok(rsp) => {
//                     info!("successfully attached integration");
//                     Ok(rsp)
//                 },
//                 Err(e) => {
//                     error!("failed to attach integration: {:?} ", e);
//                     Err(e)
//                 }
//             }; // will this cause a panic?
        

//         // TODO: once the integration is created, can delete old existing integrations
// }
        
    
//         //TODO fix this logic in case route gets created but the integration did not
//         // TODO delete existing integration? only after successfully put the new integration? 
//         // should only have one integration at a time

//     match std::fs::remove_file(&filename) {
//         Ok(_) => (),
//         Err(e) => error!("failed to delete file {}, {}", &filename, e)
//     };
    
//     // TODO: no longer hardcode the execution role arn
//     // TODO if failed deployment, update to failed status
//     update_deployment_status(job.deployment_id.as_str(), "SUCCEEDED", pool.clone()).await?;
    
//     Ok(())
// }


// async fn find_route_id(api_id: &str, route_key: &str, apig: &apigateway::Client) -> Result<String, Error> {
//     let mut found_route_id: bool = false;
//     let mut route_id: Option<String> = None;
//     let mut route_id: Option<String>;
//         // TODO optimize this. right now taking too long cuz iterating through every single route? 
//     let mut next_token: Option<String> = None;
//     loop {
//         let rsp = match next_token {
//             Some(token) => apig.get_routes()
//                 .api_id(api_id)
//                 .next_token(token)
//                 .send()
//                 .await?,
//             None => apig.get_routes()
//                 .api_id(api_id)
//                 .send()
//                 .await?
//         };

//         let routes = rsp.items().unwrap();
//         info!("found {} routes", routes.len());

//         for route in routes {
//             if found_route_id == false {

//                 if route.route_key().unwrap().eq(route_key) {
//                     found_route_id = true;
//                     info!("found route id!");
//                     // route_id = Some(route.route_id().unwrap().to_string());
//                     return Ok(route.route_id().unwrap().to_string());
//                 }
//             }
//         }

//         if rsp.next_token().is_none() {
//             return Err(error::Error::GeneralError("couldn't find route id".to_string()));
//             // finished iterating through all the routes and didn't find a matching one
//         } else {
//             next_token = Some(rsp.next_token().unwrap().to_string());
//         }
//     }
//     return Err(error::Error::GeneralError("couldn't find route id".to_string())); 

//     // if route_id.clone().is_some() {
//     //     return Ok(route_id.unwrap());
//     // } else {
//     //     return Err(error::Error::GeneralError("couldn't find route id".to_string()))

//     // }
// }

// // TODO: pre-pend user id to the function name so that it's a unique namespace? for now, it's ok
// // since we are using deployment id and that should be a uuidd
// fn run_cmd(mut cmd: &Command) { 
//     assert!(
//         cmd
//             .output()
//             .expect("command failed to start")
//             .status
//             .success()
//         ); 
// }

// // TODO: how to make it so that we aren't re-downloading it every time? 
// // idea: if we detect that the dependencies didn't change, how to use the same directory as before? 
// // Q: can we zip up the dependencies directory separately? alternatively, bundle it as a separate lambda layer
// // or is there a cache we can hit?
// fn zip(job_id, filename: &str, code: &str) -> zip::result::ZipResult<()> {
// // first, create the job directory and download all dependencies into a virtualenv

    // let tmp_dir = Path::new("/tmp");
    // let mut job_dir = tmp_dir.join(job_id);

    // assert!(
    //     Command::new("virtualenv")
    //     .arg("-p")
    //     .arg("python3")
    //     .arg(&job_dir)
    //         .output()
    //         .expect("virtualenv failed")
    //         .status
    //         .success()
    //     );

// // TODO get list of packages and store in dependencies variable
// // let dependencies = ["requests", "numpy"];
// // TODO: pipe the dependencies into a requirements.txt file
//     let dependencies = [];
//     let mut pip_cmds = "".to_string();
//     for dependency in dependencies {
//         pip_cmds.push_str(" && pip3 install ");
//         pip_cmds.push_str(dependency);
//     }

//     let mut full_cmd = "source ".to_string();
//     full_cmd.push_str(job_dir.join("bin").join("activate").to_str().unwrap());
//     if ! dependencies.is_empty() {
//         full_cmd.push_str(pip_cmds.as_str());
//     }

//     // if ! dependencies.is_empty() {
//     //     full_cmd.push_str("pip3 install -r requirements.txt");
//     // }

//     full_cmd.push_str(" && deactivate");

//     info!("full cmd: {:?}", full_cmd);


//     // Q: if this fails how will we know? will panic? how to prevent panicking? do we recover after panicking?
//     // Q: how to pip install multiple files? is calling pip multiple times slower? 
//     let output = Command::new("bash")
//         .arg("-c")
//         .arg(full_cmd)
//         .output()
//         .expect("Failed to pip install");

//     println!("status: {}", output.status);
//     println!("stdout: {}", String::from_utf8_lossy(&output.stdout));
//     println!("stderr: {}", String::from_utf8_lossy(&output.stderr));

//     // copy the lambda function to the tmp job directory, then run zip, then try uploading to see if we can hit from lambda

//     std::fs::copy(from, to)
    // uncomment this
    // match std::fs::remove_dir_all(&job_dir) {
    //     Ok(_) => (),
    //     Err(e) => error!("failed to delete directory {:?}, {}", &job_dir, e)
    // };


// // create the lambda function file


//     let path = std::path::Path::new("/tmp").join(job_id).join(filename);
//     let file = std::fs::File::create(&path).unwrap();
//     let mut zip = zip::ZipWriter::new(file);

// // Q: how to zip up the new directory
//     let options = FileOptions::default()
//         .compression_method(zip::CompressionMethod::Stored)
//         .unix_permissions(0o755);
    
//     zip.start_file("lambda_function.py", options)?;

//     zip.write_all(code.as_bytes())?;

//     zip.finish()?;

//     Ok(())
// }

// async fn get_latest_deployment
// (action_id: &str, pool: Pool<MySql>) -> Result<Deployment, error::Error> {
//     return match sqlx::query_as::<_, DeploymentVec>("SELECT * FROM Deployment WHERE actionId = ? order by createdAt desc limit 1")
//         .bind(action_id)
//         .fetch_one(&pool)
//         .await {
//             Ok(rsp) => {
//                 let deployment = Deployment {
//                     id: String::from_utf8(rsp.id).unwrap(),
//                     createdAt: rsp.createdAt,
//                     updatedAt: rsp.updatedAt,
//                     actionId: String::from_utf8(rsp.actionId).unwrap(),
//                     status: String::from_utf8(rsp.status).unwrap()
//                 };
//                 Ok(deployment)
//             }, 
//             Err(sqlx::Error::RowNotFound {}) => Err(error::Error::GetLatestDeploymentError("no deployments found".to_string())),
//             Err(e) => Err(error::Error::GetLatestDeploymentError("error fetching from db".to_string()))
//         };

    

 
// }


// async fn get_action(action_id: &str, pool: Pool<MySql>) -> Result<Action, anyhow::Error> {
//     let actionVec = sqlx::query_as::<_, ActionVec>("SELECT * FROM Action WHERE id = ?")
//         .bind(action_id)
//         .fetch_one(&pool)
//         .await?;

//     let action = Action {
//         id: String::from_utf8(actionVec.id).unwrap(),
//         createdAt: actionVec.createdAt,
//         updatedAt: actionVec.updatedAt,
//         name: String::from_utf8(actionVec.name).unwrap(), 
//         r#type: String::from_utf8(actionVec.r#type).unwrap(), 
//         code: actionVec.code.map(|x| String::from_utf8(x).unwrap()), 
//         appId: String::from_utf8(actionVec.appId).unwrap(), 
//         restVerb: actionVec.restVerb.map(|x| String::from_utf8(x).unwrap()), 
//         description: actionVec.description.map(|x| String::from_utf8(x).unwrap()), // is this the right way?
//         path: actionVec.path.map(|x| String::from_utf8(x).unwrap()) 
//     };

//     Ok(action)
// }

// async fn get_app(app_id: &str, pool: Pool<MySql>) -> Result<App, anyhow::Error> {
//     // TODO delete
//     // const id: &str = "qw2NsTR3JAgmfE"; // action id
//     let appVec = sqlx::query_as::<_, AppVec>("SELECT * FROM App WHERE id = ?")
//         .bind(app_id)
//         .fetch_one(&pool)
//         .await?;

//     let app = App {
//         id: String::from_utf8(appVec.id).unwrap(),
//         createdAt: appVec.createdAt,
//         updatedAt: appVec.updatedAt,
//         name: String::from_utf8(appVec.name).unwrap(), 
//         userId: String::from_utf8(appVec.userId).unwrap(), 
//     };

//     Ok(app)
// }

// async fn get_user(user_id: &str, pool: Pool<MySql>) -> Result<User, anyhow::Error> {
//     // TODO delete
//     // const id: &str = "qw2NsTR3JAgmfE"; // action id
//     let user_vec = sqlx::query_as::<_, UserVec>("SELECT * FROM User WHERE id = ?")
//         .bind(user_id)
//         .fetch_one(&pool)
//         .await?;

//     let user = User {
//         id: String::from_utf8(user_vec.id).unwrap(),
//         // createdAt: userVec.createdAt,
//         // updatedAt: userVec.updatedAt,
//         // email: userVec.email,
//         // familyName: userVec.familyName.map(|x| String::from_utf8(x)),
//         // givenName: , 
//         // userId: String::from_utf8(appVec.userId).unwrap(), 
//     };

//     Ok(user)
// }

// async fn update_deployment_status(id: &str, status: &str, pool: Pool<MySql>) -> Result<(), anyhow::Error>  {
//     let rsp = match sqlx::query!(
//         r#"
// UPDATE Deployment
// SET status = 'SUCCEEDED'
// WHERE id = ?
//         "#,
//         id,
//     )
//     .execute(&pool)
//     .await {
//         Ok(rsp) => {
//             if rsp.rows_affected() > 0 {
//                 info!("successfully updated deployment status");
//             } else {
//                 error!("failed to update 1 row in Deployment"); // TODO fix - this shouldn't really return ok
//             }
//             Ok(rsp)
//         },
//         Err(e) => {
//             error!("failed to update deployment status: {:?} ", e);
//             Err(e)
//         }
//     };

//     Ok(())
// }

// // TODO support case where deployment is aborted because of concurrent
