# Frequently Asked Questions

## Can I run multiple Sync Agents on the same service cluster?

Yes, absolutely, however you must configure them properly:

A given `PublishedResource` must only ever be processed by a single Sync Agent Pod. The Helm chart
configures leader-election by default, so you can scale up to have Pods on stand-by if needed.

By default the Sync Agent will discover and process all `PublishedResources` in your cluster. Use
the `--published-resource-selector` (`publishedResourceSelector` in the Helm values.yaml) to
restrict an Agent to a subset of published resources.

## Can I synchronize multiple kcp setups onto the same service cluster?

Only if you have distinct API groups (and therefore also distinct `PublishedResources`) for them.
You cannot currently publish the same API group onto multiple kcp setups. See issue #13 for more
information.
