# Conduit Connector Salesforce

## General

The Salesforce plugin is one of [Conduit](https://github.com/ConduitIO/conduit) builtin plugins.
It currently provides only source Salesforce connector, allowing for receiving Salesforce changes events in a Conduit pipeline.

## How to build it

Run `make`.

## Source

The Source connector subscribed to given topic and listens for events published by Salesforce.

### Configuration Options

| name            | description                                                                                                                                                   | required | default |
|-----------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `environment`   | Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com -> `MyDomainName`) or `sandbox` for test environment. | true     |         |
| `clientId`      | OAuth Client ID (Consumer Key).                                                                                                                               | true     |         |
| `clientSecret`  | OAuth Client Secret (Consumer Secret).                                                                                                                        | true     |         |
| `username`      | Username.                                                                                                                                                     | true     |         |
| `password`      | Password.                                                                                                                                                     | true     |         |
| `securityToken` | Security token as described here: https://help.salesforce.com/s/articleView?id=sf.user_security_token.htm&type=5.                                             | false    |         |
| `pushTopicName` | The name of the Push Topic to listen to. This value will be prefixed with `/topic/`.                                                                          | true     |         |
| `keyField`      | The name of the Response's field that should be used as a Payload's Key. Empty value will set it to `nil`.                                                    | false    | Id      |

## Destination

Not supported.

## Testing

Run `make test` to run all the unit and integration tests.

## References

- https://developer.salesforce.com/docs/apis#browse
- https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/intro_stream.htm
- https://docs.cometd.org/current7/reference/#_concepts
