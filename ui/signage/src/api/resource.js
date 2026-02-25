import {grpcWebEndpoint} from '@/api/config.js';

/**
 * Closes any open server streams associated with the given resource.
 * You should call closeResource on any ResourceValue or ResourceCollection that you no longer need the value for.
 *
 * @param {RemoteResource<any>} resource
 */
export function closeResource(resource) {
  // todo: check if grpc streams have a close or cancel method
  //  The type says cancel, but our code said close.
  if (resource?.stream?.cancel) resource.stream.cancel();
  if (resource?.stream?.close) resource.stream.close();
  if (resource?.value) resource.value = null;
  if (resource?.updateTime) resource.updateTime = null;
}

/**
 * Sets a successful value on the given resource and resets any error or loading state.
 *
 * @param {ResourceValue<V, M>} resource
 * @param {V} val
 * @template V,M
 */
export function setValue(resource, val) {
  resource.loading = false;
  resource.streamError = null;
  resource.value = val;
  resource.updateTime = new Date();
}

/**
 * Add, update, or remove an entry in the resource based on the given change.
 * Change should be an instance of the Change type from a PullFoo Smart Core request.
 * Pull RPCs define the response type with an embedded Change message,
 * the Change in pull methods on collection resources looks like the {@link CollectionChange} type.
 *
 * @param {ResourceCollection<V, M>} resource
 * @param {CollectionChange<V,M>} change
 * @param {function(V):string} idFunc A function that maps from the resource value AsObject type to a unique ID
 * @template V,M
 */
export function setCollection(resource, change, idFunc) {
  resource.loading = false;
  resource.streamError = null;
  const oldV = change.getOldValue()?.toObject();
  const newV = change.getNewValue()?.toObject();
  if (newV) {
    if (!resource.value) resource.value = {};
    resource.value[idFunc(newV)] = newV;
  } else if (oldV) {
    if (resource.value) {
      delete(resource.value[idFunc(oldV)]);
    }
  }
  resource.updateTime = change.getChangeTime().toDate();
}

/**
 * Set properties on resource to indicate that an error occurred.
 *
 * @param {RemoteResource<any,any>} resource
 * @param {Error} err
 * @param {string} name
 */
export function setError(resource, err, name = '') {
  const rErr = /** @type {RemoteError} */ {
    name,
    error: err
  };
  resource.loading = false;
  resource.streamError = rErr;
  resource.updateTime = new Date();
}

/**
 * Execute a PullFoo type RPC against a remote service that follows Smart Core patterns.
 * The actual execution of the RPC is encapsulated in the newStream argument, this function
 * receives the endpoint to connect to and should return the grpc-web ClientReadableStream
 * returned by the service client.
 *
 * This function will attempt to keep the stream open, recalling newStream if a non-fatal error occurs.
 * Manually closing the stream, typically via {@link closeResource}, will abort this retry logic.
 *
 * The value (data) returned by the stream is _NOT_ set on the given resource, this is the callers responsibility.
 * Typically this is done by subscribing to the streams data event and calling {@link setValue} or
 * {@link setCollection}.
 * Loading status and errors are recorded in the given resource.
 *
 * Typically this function is not used directly by components,
 * instead prefer using specific API helpers in the sc or ui sub-folders.
 *
 * @example Value Resource
 * const onOffValue = reactive(newResourceValue()); // used in Vue components
 * pullResource("OnOff.PullOnOff", onOffValue, endpoint => {
 *   const client = new OnOffApiPromiseClient(endpoint);
 *   const stream = client.PullOnOff(new PullOnOffRequest().setName('MyDevice'));
 *   stream.on('data', msg => {
 *     const changes = msg.getChangesList();
 *     for (const change of changes) {
 *       setValue(onOffValue, change.getOnOff().toObject());
 *     }
 *   });
 *   return stream;
 * });
 *
 * @example Collection Resource
 * const publicationsCollection = reactive(newResourceCollection()); // used in Vue components
 * pullResource('Publication.PullPublications', resource, endpoint => {
 *   const api = new PublicationApiPromiseClient(endpoint);
 *   const stream = api.pullPublications(new PullPublicationsRequest().setName('MyDevice'));
 *   stream.on('data', msg => {
 *     const changes = msg.getChangesList();
 *     for (const change of changes) {
 *       setCollection(publicationsCollection, change, v => v.id);
 *     }
 *   });
 *   return stream;
 * });
 *
 * @example Vue Component
 * <template>
 *   <v-progress-spinner v-if="value.loading" />
 *   <span v-else>{{ name }} is {{ state }}</span>
 *   <span v-if="value.error">{{ value.error }}</span>
 * </template>
 * <script setup>
 *   // imports
 *   const name = ref('MyDevice');
 *   const resource = reactive(newResourceValue());
 *   const state = computed(() => {
 *     switch (resource.value?.state) {
 *       case OnOff.State.ON: return 'on';
 *       case OnOff.State.OFF: return 'off';
 *       default: return 'unknown';
 *     }
 *   });
 *
 *   watch(name, () => {
 *     pullResource("OnOff.PullOnOff", resource, endpoint => {
 *       const client = new OnOffApiPromiseClient(endpoint);
 *       const stream = client.PullOnOff(new PullOnOffRequest().setName(name.value));
 *       stream.on('data', msg => {
 *         const changes = msg.getChangesList();
 *         for (const change of changes) {
 *           setValue(resource, change.getOnOff().toObject());
 *         }
 *       });
 *       return stream;
 *     });
 *   }, {immediate: true});
 *   onUnmounted(() => {
 *     closeResource(value);
 *   });
 * </script>
 *
 * @param {string} logPrefix
 * @param {RemoteResource<O, T>} resource
 * @param {StreamFactory<T>} newStream
 * @template T,O
 */
export function pullResource(logPrefix, resource, newStream) {
  const doPull = (retryDelayMs = 1000) => {
    let retryCalled = false;
    const retry = () => {
      if (retryCalled) return;
      retryCalled = true;

      const handle = setTimeout(() => {
        const delay = Math.max(1000, Math.min(retryDelayMs * 2, 15 * 1000));
        doPull(delay);
      }, retryDelayMs);
      // fake stream we use to cancel the timeout if this component is disposed.
      resource.stream = {
        cancel() {
          clearTimeout(handle);
        }
      };
    };

    const address = grpcWebEndpoint();
    Promise.resolve(address)
        .then((endpoint) => {
          const stream = newStream(endpoint);
          resource.stream = stream;
          stream.on('data', (r) => {
            retryDelayMs = 1000; // if we were successful, we reset the retry delay
            resource.lastResponse = r;
          });
          stream.on('error', (err) => {
            setError(resource, err, logPrefix);
            retry();
          });
          stream.on('end', () => {
            retry();
          });
        })
        .catch((err) => {
          setError(resource, err, logPrefix);
          retry();
        });
  };

  doPull(0);
}

/**
 * Execute a non-streaming RPC against the globally configured endpoint.
 * The returned result or error and other metadata about the request is stored in tracker, which is typically reactive.
 * The actual RPC is defined by the caller in the action argument,
 * this function is called with the globally defined server endpoint and should return a Promise that resolves to the
 * grpc-web message object returned by the RPC.
 *
 * The AsObject representation of the response from the RPC is returned upon success, this value is also recorded in
 * the given ActionTracker.value. If the RPC fails this error will be used to reject the returned Promise.
 *
 * Typically this function is not used directly by components,
 * instead prefer using specific API helpers in the sc or ui sub-folders.
 *
 * @example
 * const updateOnOffAction = reactive(newActionTracker()); // use by Vue components
 * trackAction("OnOff.UpdateOnOff", updateOnOffAction, endpoint => {
 *   const client = new OnOffApiPromiseClient(endpoint);
 *   return client.UpdateOnOff(new UpdateOnOffRequest()
 *     .setName('MyDevice')
 *     .setOnOff(new OnOff().setState(OnOff.State.ON))
 *   );
 * });
 *
 * @example Vue Component Use
 * <template>
 *   <v-btn @click="turnOn" :loading="action.loading" :color="action.error ? 'red' : undefined">Turn On</v-btn>
 * </template>
 * <script setup>
 *   // imports
 *   const props = defineProps(['name']);
 *   const action = reactive(newActionTracker());
 *   function turnOn() {
 *     trackAction("OnOff.UpdateOnOff", updateOnOffAction, endpoint => {
 *       const client = new OnOffApiPromiseClient(endpoint);
 *       return client.UpdateOnOff(new UpdateOnOffRequest()
 *         .setName(props.name)
 *         .setOnOff(new OnOff().setState(OnOff.State.ON))
 *       );
 *     })
 *       .catch(err => console.error('error turning on', err);
 *   }
 * </script>
 *
 * @param {string} logPrefix
 * @param {ActionTracker<V>} tracker
 * @param {Action<V, M>} action
 * @return {Promise<V>}
 * @template V, M
 */
export async function trackAction(logPrefix, tracker, action) {
  tracker.loading = true;
  const endpoint = await grpcWebEndpoint();
  try {
    const msg = await action(endpoint);
    const value = msg.toObject();
    tracker.response = value;
    tracker.error = null;
    return value;
  } catch (err) {
    const rErr = /** @type {RemoteError} */ {
      name: logPrefix,
      error: err
    };
    tracker.error = rErr;
    throw err;
  } finally {
    tracker.loading = false;
  }
}

/**
 * Returns a blank ActionTracker with all properties populated with their defaults.
 *
 * @return {ActionTracker<V>}
 * @template V
 * @see trackAction
 */
export function newActionTracker() {
  return {
    loading: false,
    response: null,
    error: null,
    duration: 0
  };
}

/**
 * Returns a blank ResourceValue with all properties populated with their defaults.
 * This is useful in Vue components, specifically data properties or via reactive.
 *
 * @return {ResourceValue<V, M>}
 * @template V,M
 * @see pullResource
 */
export function newResourceValue() {
  return {
    loading: false,
    stream: null,
    streamError: null,
    updateTime: null,
    value: null
  };
}

/**
 * Returns a blank ResourceValue with all properties populated with their defaults.
 * This is useful in Vue components, specifically data properties or via reactive.
 *
 * @return {ResourceCollection<V, M>}
 * @template V,M
 * @see pullResource
 */
export function newResourceCollection() {
  return {
    loading: false,
    stream: null,
    streamError: null,
    lastResponse: null,
    updateTime: null,
    value: {}
  };
}
