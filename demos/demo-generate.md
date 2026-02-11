# Generating Contracts from OpenAPI Specs

*2026-02-11T22:19:40Z*

Writing contract YAML from scratch means reading the API docs and hand-building every interaction. The `generate` command reads an OpenAPI spec and produces starter contracts with sensible defaults and matching rules.

## The OpenAPI Spec

Here's a petstore API spec with several endpoints, path parameters, enums, patterns, and an allOf composition:

```bash
cat testdata/openapi/petstore.yaml
```

```output
openapi: "3.0.3"
info:
  title: Petstore
  version: "1.0.0"

paths:
  /pets:
    get:
      summary: List all pets
      operationId: listPets
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
      responses:
        "200":
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Pet"
    post:
      summary: Create a pet
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NewPet"
      responses:
        "201":
          description: Pet created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
        "400":
          description: Invalid input

  /pets/{petId}:
    get:
      summary: Get a pet by ID
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            example: 42
      responses:
        "200":
          description: A single pet
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
        "404":
          description: Pet not found

  /pets/{petId}/health:
    get:
      summary: Get pet health record
      operationId: getPetHealth
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: Health record
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/HealthRecord"

components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
        - status
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
          example: "Fido"
        tag:
          type: string
        status:
          type: string
          enum:
            - available
            - pending
            - sold
        email:
          type: string
          format: email
          pattern: "^[^@]+@[^@]+$"

    NewPet:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        tag:
          type: string

    HealthRecord:
      allOf:
        - type: object
          required:
            - petId
          properties:
            petId:
              type: integer
        - type: object
          required:
            - vaccinated
          properties:
            vaccinated:
              type: boolean
            lastCheckup:
              type: string
              format: date-time
```

## Generating a Contract

Use `--dry-run` to preview the generated contract without writing any files. The consumer name defaults to "my-service" and the provider name is derived from the spec title:

```bash
./accord generate --dry-run --consumer order-service testdata/openapi/petstore.yaml
```

```output
# order-service--petstore.yaml
accord: "0.1"
consumer:
    name: order-service
provider:
    name: petstore
interactions:
    - description: List all pets
      request:
        method: GET
        path: /pets
        headers: {}
        query: {}
        body: null
      response:
        status: 200
        headers: {}
        body:
            - id: 0
              name: Fido
              status: available
      matching_rules:
        $.body:
            match: type
    - description: Create a pet
      request:
        method: POST
        path: /pets
        headers: {}
        query: {}
        body:
            name: string
      response:
        status: 201
        headers: {}
        body:
            id: 0
            name: Fido
            status: available
      matching_rules:
        $.body.id:
            match: type
        $.body.name:
            match: type
        $.body.status:
            match: type
    - description: Get a pet by ID
      request:
        method: GET
        path: /pets/42
        headers: {}
        query: {}
        body: null
      response:
        status: 200
        headers: {}
        body:
            id: 0
            name: Fido
            status: available
      matching_rules:
        $.body.id:
            match: type
        $.body.name:
            match: type
        $.body.status:
            match: type
    - description: Get pet health record
      request:
        method: GET
        path: /pets/1/health
        headers: {}
        query: {}
        body: null
      response:
        status: 200
        headers: {}
        body:
            petId: 0
            vaccinated: false
      matching_rules:
        $.body.petId:
            match: type
        $.body.vaccinated:
            match: type

```

Several things to notice: path parameters are substituted with example values (`{petId}` became `42` from the spec's example, or `1` as the default for integers), only required fields appear in request and response bodies, the `allOf` composition in HealthRecord is merged into a flat object, and matching rules are set to `type` by default.

## Filtering Endpoints

Use `--endpoints` with a glob pattern to generate contracts for specific paths only. This is useful when your service only consumes a subset of a provider's API:

```bash
./accord generate --dry-run --endpoints '/pets' --consumer order-service testdata/openapi/petstore.yaml
```

```output
# order-service--petstore.yaml
accord: "0.1"
consumer:
    name: order-service
provider:
    name: petstore
interactions:
    - description: List all pets
      request:
        method: GET
        path: /pets
        headers: {}
        query: {}
        body: null
      response:
        status: 200
        headers: {}
        body:
            - id: 0
              name: Fido
              status: available
      matching_rules:
        $.body:
            match: type
    - description: Create a pet
      request:
        method: POST
        path: /pets
        headers: {}
        query: {}
        body:
            name: string
      response:
        status: 201
        headers: {}
        body:
            id: 0
            name: Fido
            status: available
      matching_rules:
        $.body.id:
            match: type
        $.body.name:
            match: type
        $.body.status:
            match: type

```

The pattern `/pets` matched only the two operations on `/pets` exactly, excluding `/pets/{petId}` and `/pets/{petId}/health`.

## Writing Files

Without `--dry-run`, the command writes files to disk. The filename follows the `{consumer}--{provider}.yaml` convention:

```bash
mkdir -p /tmp/accord-demo && ./accord generate --output-dir /tmp/accord-demo --consumer order-service testdata/openapi/minimal.yaml && ls /tmp/accord-demo/
```

```output
wrote /tmp/accord-demo/order-service--minimal-service.yaml
order-service--minimal-service.yaml
```

## Round-Trip: Generate then Lint

Generated contracts are valid by construction. We can prove this by linting the output:

```bash
mkdir -p /tmp/accord-demo2 && ./accord generate --output-dir /tmp/accord-demo2 --consumer test-svc testdata/openapi/petstore.yaml && ./accord lint /tmp/accord-demo2/test-svc--petstore.yaml && echo 'exit code: 0'
```

```output
wrote /tmp/accord-demo2/test-svc--petstore.yaml
exit code: 0
```

No lint errors - the generated contract passes all validation rules. From here you can edit the contract to tighten matching rules, add headers, or remove interactions you don't need.
