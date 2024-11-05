# Conduit Connector Salesforce

## General

The Salesforce plugin is one of [Conduit](https://github.com/ConduitIO/conduit) plugins.
It currently provides only source Salesforce connector, allowing for receiving Salesforce changes events in a Conduit pipeline.

## How to build it

Run `make`.

## Source

The Source connector subscribes to Salesforce platform events and queries published events in real time.

## Destination

The Destination connector subscribes to Salesforce platform events and publishes incoming records to the event in real time.

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


### Configuration Options

With the above set up followed, you can begin configuring the source or destination connector. Refer to the table below on which values to set. 

#### Source


| name              | description                                                                                                                                                                                                                                    | required | default |
|-------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `oauthEndpoint`     | Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com ) |   | `true`  |
| `clientId`        | OAuth Client ID (Consumer Key)           | `true`   |         |
| `clientSecret`    | OAuth Client Secret (Consumer Secret)      | `true`   |         |
| ~~`topicName`~~ | Event topic name for your event (e.g: /event/Accepted_Quote__e) **Deprecated: use `topicNames` instead**  |	`false` ||
| `topicsNames`        | One or multiple comma separated topic names the source will subscribe to (e.g */event/Test__e, /event/Test2__e*).  | `true`   ||
| `retryCount`        | Number of times the connector will retry is the connection to a topic breaks.  | `false`   |    `10`     |
| `replayPreset`        | The position from which connector will start reading events, either 'latest' or 'earliest'. Latest will pull only newly created events, and earlies will pull any events that are currently in the topic.  | `false`   |    `earliest`     |
| `pollingPeriod`        | The client event polling interval between each data read on the topic. | `false`   |    `100ms`     |
| `insecureSkipVerify`        | Disables certificate validation. | `false`   |   `false`      |


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


## Destination

| name              | description                                                                                                                                                                                                                                    | required | default |
|-------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `oauthEndpoint`     | Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com ) |   | `true`  |
| `clientId`        | OAuth Client ID (Consumer Key)           | `true`   |         |
| `clientSecret`    | OAuth Client Secret (Consumer Secret)      | `true`   |         |
| `topicName` | Event topic name for your event (e.g: /event/Accepted_Quote__e)  |	`true` ||
| `retryCount`        | Number of times the connector will retry is the connection to a topic breaks.  | `false`   |    `10`     |
| `insecureSkipVerify`        | Disables certificate validation. | `false`   |   `false`      |


## Testing

Run `make test` to run all the unit and integration tests.

## References

- https://developer.salesforce.com/docs/apis#browse
- https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/intro_stream.htm
- https://docs.cometd.org/current7/reference/#_concepts
