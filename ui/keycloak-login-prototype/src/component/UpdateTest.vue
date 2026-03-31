<script setup>
import {OnOffApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_grpc_web_pb';
import {OnOff, UpdateOnOffRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb';
import {ref} from 'vue';

const props = defineProps(['token'])

const data = ref(OnOff.State.STATE_UNSPECIFIED)
const error = ref(null)

const client = new OnOffApiPromiseClient('https://localhost:8000', null, null)

async function updateTest() {
  const token = props.token
  if (token === undefined || token === null) {
    window.alert('get token first')
    return
  }

  const request = new UpdateOnOffRequest()
      .setName('dev-1')
      .setOnOff(new OnOff().setState(data.value))

  try {
    await client.updateOnOff(request, {
      'Authorization': 'Bearer ' + token,
    })
    error.value = null
  } catch (e) {
    error.value = e
  }
}

</script>

<template>
  <h2>Update Test</h2>
  <h3>Value to Write</h3>
  <pre>
{
<label>
  "state": <select v-model="data">
  <option :value="OnOff.State.STATE_UNSPECIFIED">UNSPECIFIED (0)</option>
  <option :value="OnOff.State.ON">ON (1)</option>
  <option :value="OnOff.State.OFF">OFF (2)</option>
  </select>
</label>
}
  </pre>

  <div>
    <button @click="updateTest">OnOffApi.UpdateOnOff</button>
  </div>
  <div v-if="error !== null">
    <h3>Error</h3>
    <pre>{{ error }}</pre>
  </div>
</template>
