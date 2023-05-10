# chatgpt-api-server

## API 

The API has only one endpoint located at `/prompt`, which accepts an HTTP POST request with a plain text body.

The response sent back from the server is also in plain text. Ideally, the response should be in JSON format, but in some cases, the response may be a combination of plain text and JSON. In such cases, the client should parse the response to extract the JSON data.

If there is no JSON content inside the response body, the client should treat it as an error. This means that the server did not return the expected data and the client should handle this error accordingly.

## Configuration

You need to provide a list of environment variables as well as an instruction file.
The following environment variables are required:

Env vars:
```
GPT_API_KEY=<key>
GPT_INSTRUCTION_FILE_PATH=<path/to/instruction-file> || GPT_INSTRUCTION_TEXT=<insturction>
```

The instruction file provides the necessary context and database information that will be used to answer client requests, as well as instructions to run ChatGPT as a backend web server.

### Create an OpenAI API KEY

Create an account and request your OpenAPI key in https://beta.openai.com/account/api-keys

## Docker

The easiest way to use the Dubai Backend is to run it inside a Docker container.

### Build

To build the Docker image, run the following command in the root directory of this project:

```
docker build . -t <name>:<tag>
```

### Run

1. Create an envfile and add the required environment variables.

2. Mount the instruction file to the container with the `-v` flag. The `GPT_INSTRUCTION_FILE_PATH` environment variable should specify the file path inside the container.

3. Pass the envfile to the container with the `--env-file` option.

Use the following command to run the Docker container:
```
docker run --restart always -td -p 8080:8080 -v /local/path/to/instruction-file:/container/path --env-file /path/to/envfile <name>:<tag>
```

### Example

envfile
```
GPT_API_KEY=<API_KEY>
GPT_INSTRUCTION_FILE_PATH=/tmp/instruction.txt
```

instruction.txt
```
CONTENT
```

Run
```
docker run --restart always -td -p 8080:8080 -v instruction.txt:/tmp/instruction.txt --env-file envfile backend:latest
```
