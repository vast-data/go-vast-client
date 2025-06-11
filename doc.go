/*
Package vast_client provides a typed and convenient interface to interact with the VAST Data REST API.

It wraps raw HTTP operations in a structured API, exposing high-level methods to manage VAST resources
like views, volumes, quotas, snapshots, and more. Each resource is available as a sub-client that
supports common CRUD operations (List, Get, GetById, Create, Update, Delete, etc.).

The main entry point is the VMSRest client, which is initialized using a VMSConfig configuration struct.
This configuration allows customization of connection parameters, credentials (username/password or token),
SSL behavior, request timeouts, and request/response hooks.
*/
package vast_client
