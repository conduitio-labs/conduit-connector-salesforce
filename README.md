# Conduit Connector Salesforce

## General

The Salesforce plugin is one of [Conduit](https://github.com/ConduitIO/conduit) plugins.
It currently provides only source Salesforce connector, allowing for receiving Salesforce changes events in a Conduit pipeline.

## How to build it

Run `make`.

## Source

The Source connector subscribed to given topic and listens for events published by Salesforce.

### Configuration Options

| name              | description                                                                                                                                                             | required | default |
|-------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `environment`     | Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com -> `MyDomainName`) or `sandbox` for test environment.           | `true`   |         |
| `clientId`        | OAuth Client ID (Consumer Key).                                                                                                                                         | `true`   |         |
| `clientSecret`    | OAuth Client Secret (Consumer Secret).                                                                                                                                  | `true`   |         |
| `username`        | Username.                                                                                                                                                               | `true`   |         |
| `password`        | Password.                                                                                                                                                               | `true`   |         |
| `securityToken`   | Security token as described here: https://help.salesforce.com/s/articleView?id=sf.user_security_token.htm&type=5.                                                       | `false`  |         |
| `pushTopicsNames` | The comma-separated list of names of the Push Topics to listen to. All values will be prefixed with `/topic/`. All Topics have to exist for connector to start working. | `true`   |         |
| `keyField`        | The name of the Response's field that should be used as a Payload's Key. Empty value will set it to `nil`.                                                              | `false`  | `Id`    |

### Step-by-step configuration example

There are a couple of steps that need to be done to start working with Salesforce connector.

1. Log in into Your Salesforce account, e.g. https://my-demo-app.my.salesforce.com. The environment is `my-demo-app`.
2. First, if not already done, You need to create connected app and enable OAuth: [Enable OAuth Settings for API Integration](https://help.salesforce.com/s/articleView?id=sf.connected_app_create_api_integration.htm&type=5). Successfully configured app example can be seen below:
    ![Connected App example](docs/connect_and_configure_app.png)
3. Copy **Consumer Key** and **Consumer Secret**. If You need these values once again You can always find them in _Setup -> Apps -> App Manager_, find app on the list and choose _View_ option.
    ![View OAuth tokens](docs/view_oauth_tokens.png)
4. You may need to configure **Security Token** for Your account. For more details follow instructions: [Reset Your Security Token](https://help.salesforce.com/s/articleView?id=sf.user_security_token.htm&type=5).
5. When all credentials are done, next You need to [create push topics](https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/code_sample_interactive_vfp_create_pushtopic.htm).
    Configure topics to Your needs specifying query and notifications behaviour for them.
6. Once done, You can begin with configuring the connector:
   1. Use Step 1 environment value as `environment` config, e.g. `my-demo-app`.
   2. Use Step 3 **Consumer Key** value as `clientId` config.
   3. Use Step 3 **Consumer Secret** value as `clientSecret` config.
   4. Use Step 1 username and password credentials as values for `username` and `password` config.
   5. If required for Your account, use Step 4 **Security Token** value as `securityToken` config.
   6. Use Step 5 Topic Names as a comma-separated values for `pushTopicsNames` config, e.g. `TaskUpdates,ContactUpdates`.
   7. Optionally, configure `keyField`. The field's with this name value will be used as a Record Key, which may be utilized by other connectors.

        Example:

        When Push Topic query is: `SELECT Id, Name FROM Task` and `keyField` is set to **Id**,

        Then for event with value `Id=123, Name=Create summary` Record's Key will be set to `123`.

        Later, this may be utilized by other connectors, e.g. [Elasticsearch connector](https://github.com/miquido/conduit-connector-elasticsearch) will create Document with ID of Record's Key when available.

## Destination

Not supported.

## Testing

Run `make test` to run all the unit and integration tests.

## References

- https://developer.salesforce.com/docs/apis#browse
- https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/intro_stream.htm
- https://docs.cometd.org/current7/reference/#_concepts
