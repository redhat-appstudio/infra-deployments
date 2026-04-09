# Smee-client component

The `smee-client` component deploys [gosmee][gs] in client mode.

This allows a cluster to consume webhooks forwarded via our Smee service.

## Webhook forwarding service

For development, use [hook.pipelinesascode.com][hpac] to create webhook
forwarding channels. Do **not** use smee.io — it does not properly preserve
webhook signatures, which causes Forgejo webhook signature validation to fail.
hook.pipelinesascode.com runs gosmee on the server side and correctly forwards
the original webhook headers and payload.

[gs]: https://github.com/chmouel/gosmee
[hpac]: https://hook.pipelinesascode.com
