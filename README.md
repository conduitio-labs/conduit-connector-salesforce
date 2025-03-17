# Conduit Connector Salesforce

<!-- readmegen:description -->

## General

The Salesforce plugin is one of [Conduit](https://github.com/ConduitIO/conduit) plugins.

## How to build it

Run `make`.

## Source

The Source connector subscribes to Salesforce platform events and queries published events in real time.

## Destination

The Destination connector publishes incoming records to Salesforce platform events in real time.

### How it works

This section describes the technical details of how the connector works.

#### PubSub API

Conduit Source and Destination Connector uses [PubSub API](https://developer.salesforce.com/docs/platform/pub-sub-api/overview) for subscribing to event data.

##### Platform Events

The connector is capable of listening to [custom Platform Events](https://developer.salesforce.com/docs/atlas.en-us.platform_events.meta/platform_events/platform_events_intro.htm).

Platform Events in Salesforce are part of the event-driven architecture, allowing for the exchange of real-time event data between applications. They enable asynchronous communication and data transfer between Salesforce and external systems, as well as among various Salesforce components. Users can define custom Platform Events based on their specific use cases, to which the connector can subscribe. Additionally, users can subscribe to already [existing events](https://developer.salesforce.com/docs/atlas.en-us.platform_events.meta/platform_events/platform_events_objects_list.htm).

For more information on how to create Platform Events, visit the following link:
[Creating Platform Events](https://developer.salesforce.com/docs/atlas.en-us.platform_events.meta/platform_events/platform_events_publish.htm)

###### Setting Platform Events from a Flow

Steps to Create Platform Events

1. **Create a Platform Event**

- Navigate to **Setup**.
- Under **Platform Tools**, go to **Integrations** > **Platform Events**.
- Create a new Platform Event and define all the fields for the event to match the desired data structure.

2. **Create a Flow**

- Go to **Process Automation** > **Flow** and create a new flow.
- Define the source of the data for the previously created event.
- Select **Record-Triggered Flow** if you want to trigger the event based on new, updated, or deleted records on an object.
- Apply any necessary transformations or additional data assignments within the flow.
- End your flow with a **Create Record** element, where you will assign field values for the created event.

#### Authentication

The connector authenticates with Salesforce using OAuth credentials. Once successfully authenticated, Salesforce returns:

- `AccessToken` - token used in Streaming API requests.
- `Instance URL` - Streaming API base URL for given credentials.

##### Connected App Configuration

1. Log in into Your Salesforce account, e.g. https://my-demo-app.my.salesforce.com. The environment is `my-demo-app`.
2. First, if not already done, You need to create connected app and enable following settings:

   Under **API (Enable OAuth Settings)**

   - Enable `Enable OAuth Settings`
   - Add the following permissions for Selected OAuth Scope:
     - `Access the Salesforce API Platform`
     - `Perform requests at any time`
   - Enable `Require Proof Key for Code Exchange (PKCE) Extension for Supported Authorization Flows`
   - Enable `Require Secret for Web Server Flow`
   - Enable `Require Secret for Refresh Token Flow`
   - Enable `Enable Client Credentials Flow`

   Under **Custom Connected App Handler**

   - On `Run As` select a user with 'API Enabled' permission privileges. (If user doesn't have them, you can add them under Permission Sets on User settings)

   ![Connected App example](docs/connected_app.png)

3. Copy **Consumer Key** and **Consumer Secret**. If You need these values once again You can always find them in _Setup -> Apps -> App Manager_, find app on the list and choose _View_ option.
   ![View OAuth tokens](docs/view_oauth_tokens.png)

4. Once the app is created, go to _Connected Apps -> Manage Connected Apps_ and edit the application.
   Under **Client Credentials Flow**
   - On `Run As` select a user with 'API Enabled' permission privileges. (If user doesn't have them, you can add them under Permission Sets on User settings)

### Error handling

The connector will report any errors at startup if authentication credentials are incorrect, which may lead to a server request rejection.

If any specified Platform Events do not exist, Salesforce will reject the subscription, resulting in connector failure.

If the `keyField` is configured but not found in the data, an error will be returned.

While the connector is operational, it may be necessary to reconnect. The server provides conditions for reconnection, but it may also terminate the connection. The connector handles these scenarios by attempting to reconnect or notifying the user of the termination.

<!-- /readmegen:description -->

### Configuration Options

With the above set up followed, you can begin configuring the source or destination connector. Refer to the table below on which values to set.

Additionally, on the destination connector, verify that the source record’s data fields align with those of the event it’s being published to. You can include or exclude fields from the data as necessary using Conduit Processors.

#### Source

<!-- readmegen:source.parameters.yaml -->

```yaml
version: 2.2
pipelines:
  - id: example
    status: running
    connectors:
      - id: example
        plugin: 'salesforce'
        settings:
          # ClientID is the client id from the salesforce app
          # Type: string
          # Required: yes
          clientID: ''
          # ClientSecret is the client secret from the salesforce app
          # Type: string
          # Required: yes
          clientSecret: ''
          # OAuthEndpoint is the OAuthEndpoint from the salesforce app
          # Type: string
          # Required: yes
          oauthEndpoint: ''
          # InsecureSkipVerify disables certificate validation
          # Type: bool
          # Required: no
          insecureSkipVerify: 'false'
          # PollingPeriod is the client event polling interval
          # Type: duration
          # Required: no
          pollingPeriod: '100ms'
          # gRPC Pubsub Salesforce API address
          # Type: string
          # Required: no
          pubsubAddress: 'api.pubsub.salesforce.com:7443'
          # Replay preset for the position the connector is fetching events
          # from, can be latest or default to earliest.
          # Type: string
          # Required: no
          replayPreset: 'earliest'
          # Number of retries allowed per read before the connector errors out
          # Type: int
          # Required: no
          retryCount: '10'
          # Deprecated: use `topicNames` instead.
          # Type: string
          # Required: no
          topicName: ''
          # TopicNames are the TopicNames the source connector will subscribe to
          # Type: string
          # Required: no
          topicNames: ''
          # Maximum delay before an incomplete batch is read from the source.
          # Type: duration
          # Required: no
          sdk.batch.delay: '0'
          # Maximum size of batch before it gets read from the source.
          # Type: int
          # Required: no
          sdk.batch.size: '0'
          # Specifies whether to use a schema context name. If set to false, no
          # schema context name will be used, and schemas will be saved with the
          # subject name specified in the connector (not safe because of name
          # conflicts).
          # Type: bool
          # Required: no
          sdk.schema.context.enabled: 'true'
          # Schema context name to be used. Used as a prefix for all schema
          # subject names. If empty, defaults to the connector ID.
          # Type: string
          # Required: no
          sdk.schema.context.name: ''
          # Whether to extract and encode the record key with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.key.enabled: 'true'
          # The subject of the key schema. If the record metadata contains the
          # field "opencdc.collection" it is prepended to the subject name and
          # separated with a dot.
          # Type: string
          # Required: no
          sdk.schema.extract.key.subject: 'key'
          # Whether to extract and encode the record payload with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.payload.enabled: 'true'
          # The subject of the payload schema. If the record metadata contains
          # the field "opencdc.collection" it is prepended to the subject name
          # and separated with a dot.
          # Type: string
          # Required: no
          sdk.schema.extract.payload.subject: 'payload'
          # The type of the payload schema.
          # Type: string
          # Required: no
          sdk.schema.extract.type: 'avro'
```

<!-- /readmegen:source.parameters.yaml -->

##### The generated payload

Connector produces [`sdk.StructuredData`](https://github.com/ConduitIO/conduit-connector-sdk/blob/main/record.go) data type with information from the received event. The payload entirely depends on the Publish Events fields.
The following data is included:

- `Key` - either `nil` when not configured or the value of payload's `keyField` field.
- `Payload` - Salesforce Publish Events result; a decoded JSON value.
- `Position` - Event's Replay ID, generated by Salesforce.
- `CreatedAt` - Event's Creation Date, generated by Salesforce.
- `Metadata.channel` - A channel name of the event.
- `Metadata.replyId` - Event's Replay ID.
- `Metadata.action` - Event's action, i.e. the reason why notification was sent. Currently, all events will be marked a created.

#### Destination

<!-- readmegen:destination.parameters.yaml -->

```yaml
version: 2.2
pipelines:
  - id: example
    status: running
    connectors:
      - id: example
        plugin: 'salesforce'
        settings:
          # ClientID is the client id from the salesforce app
          # Type: string
          # Required: yes
          clientID: ''
          # ClientSecret is the client secret from the salesforce app
          # Type: string
          # Required: yes
          clientSecret: ''
          # OAuthEndpoint is the OAuthEndpoint from the salesforce app
          # Type: string
          # Required: yes
          oauthEndpoint: ''
          # Topic is Salesforce event or topic to write record
          # Type: string
          # Required: yes
          topicName: ''
          # InsecureSkipVerify disables certificate validation
          # Type: bool
          # Required: no
          insecureSkipVerify: 'false'
          # gRPC Pubsub Salesforce API address
          # Type: string
          # Required: no
          pubsubAddress: 'api.pubsub.salesforce.com:7443'
          # Number of retries allowed per read before the connector errors out
          # Type: int
          # Required: no
          retryCount: '10'
          # Maximum delay before an incomplete batch is written to the
          # destination.
          # Type: duration
          # Required: no
          sdk.batch.delay: '0'
          # Maximum size of batch before it gets written to the destination.
          # Type: int
          # Required: no
          sdk.batch.size: '0'
          # Allow bursts of at most X records (0 or less means that bursts are
          # not limited). Only takes effect if a rate limit per second is set.
          # Note that if `sdk.batch.size` is bigger than `sdk.rate.burst`, the
          # effective batch size will be equal to `sdk.rate.burst`.
          # Type: int
          # Required: no
          sdk.rate.burst: '0'
          # Maximum number of records written per second (0 means no rate
          # limit).
          # Type: float
          # Required: no
          sdk.rate.perSecond: '0'
          # The format of the output record. See the Conduit documentation for a
          # full list of supported formats
          # (https://conduit.io/docs/using/connectors/configuration-parameters/output-format).
          # Type: string
          # Required: no
          sdk.record.format: 'opencdc/json'
          # Options to configure the chosen output record format. Options are
          # normally key=value pairs separated with comma (e.g.
          # opt1=val2,opt2=val2), except for the `template` record format, where
          # options are a Go template.
          # Type: string
          # Required: no
          sdk.record.format.options: ''
          # Whether to extract and decode the record key with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.key.enabled: 'true'
          # Whether to extract and decode the record payload with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.payload.enabled: 'true'
```

<!-- /readmegen:destination.parameters.yaml -->

## Testing

Run `make test` to run all the unit and integration tests.

## References

- https://developer.salesforce.com/docs/apis#browse
- https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/intro_stream.htm
- https://docs.cometd.org/current7/reference/#_concepts
