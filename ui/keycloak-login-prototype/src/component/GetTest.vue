<script setup>
import {OnOffApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_grpc_web_pb';
import {GetOnOffRequest} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/onoff/v1/on_off_pb';
import {reactive} from 'vue'

const props = defineProps(['token'])
const client = new OnOffApiPromiseClient('https://localhost:8000', null, null)

const result = reactive({
  data: /** @type {OnOff.AsObject} */ null,
  error: null
})

async function getTest() {
  const token = props.token
  if (token === null) {
    window.alert("get token first")
    return
  }

  result.data = null

  try {
    const res = await client.getOnOff(new GetOnOffRequest().setName('dev-1'),{
      "Authorization": "Bearer " + props.token
    })
    result.data = res.toObject()
    result.error = null
  } catch (e) {
    result.error = e
  }
}
</script>

<template>
  <h2>Get Test</h2>
  <button @click="getTest">OnOffApi.GetOnOff</button>
  <div v-if="result.data !== null">
    <h3>Response</h3>
    <pre>{{ result.data }}</pre>
  </div>
  <div v-if="result.error !== null">
    <h3>Error</h3>
    <pre>{{ result.error }}</pre>
  </div>
</template>
