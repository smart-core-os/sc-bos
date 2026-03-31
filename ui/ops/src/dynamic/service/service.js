import {newActionTracker} from '@/api/resource.js';
import {configureService} from '@/api/ui/services.js';
import {statusCodeToString} from '@/components/ui-error/util.js';
import {useOfflineEdit} from '@/composables/edit.js';
import useAuthSetup from '@/composables/useAuthSetup.js';
import {formatErrorMessage} from '@/util/error.js';
import {cloneDeep} from 'lodash';
import {computed, inject, reactive, ref} from 'vue';

/**
 * @template T
 * @param {(T) => T} [beforeEdit]
 * @param {(T) => T} [beforeSave]
 * @return {{
 *   fabAttrs: Ref<Record<string, any>>,
 *   saveAttrs: Ref<Record<string, any>>,
 *   refreshNeeded: Ref<boolean>,
 *   refreshAttrs: Ref<Record<string, any>>,
 *   saveTooltip: Ref<string>,
 *   refreshTooltip: Ref<string>,
 *   alertVisible: Ref<boolean>,
 *   alertAttrs: Ref<Record<string, any>>,
 *   loading: Ref<boolean>,
 *   remoteModel: Ref<T>,
 *   configModel: Ref<T>,
 *   'readonly': Ref<boolean>,
 *   serviceID: Ref<string | undefined>,
 *   serviceType: Ref<string | undefined>,
 *   serviceDisabled: Ref<boolean>
 * }}
 */
export function useServiceConfig(beforeEdit = (v) => v, beforeSave = (v) => v) {
  const service = inject('service.value');
  const serviceLoading = inject('service.loading');
  const serviceError = inject('service.streamError');
  const serviceName = inject('service.name');

  const doBefore = (v, fn) => {
    const v1 = cloneDeep(v);
    const v2 = fn(v1);
    if (v2 === undefined) return v1;
    return v2;
  };

  const configParsed = computed(() => doBefore(JSON.parse(service.value?.configRaw || '{}'), beforeEdit));
  const configModel = ref();
  const localParseError = ref();
  const localChanges = computed({
    get() {
      return configModel.value;
    },
    set(v) {
      if (typeof v === 'string') {
        try {
          v = JSON.parse(v);
          configModel.value = v;
          localParseError.value = null;
          return;
        } catch (e) {
          localParseError.value = e;
        }
      } else {
        localParseError.value = null;
      }
      configModel.value = cloneDeep(v);
    }
  });

  const {blockActions} = useAuthSetup();

  const {localIsChanged, remoteIsChanged, sync} = useOfflineEdit(configParsed, localChanges);
  const saveTracker = reactive(newActionTracker());
  const doSave = () => {
    const request = {
      name: serviceName.value,
      id: service.value.id,
      configRaw: JSON.stringify(doBefore(configModel.value, beforeSave))
    };
    configureService(request, saveTracker)
        .catch(() => {}); // handled via saveTracker
  };

  const saveAttrs = computed(() => ({
    disabled: Boolean(!localIsChanged.value || remoteIsChanged.value || !configModel.value || saveTracker.loading || blockActions.value || localParseError.value),
    loading: saveTracker.loading,
    text: 'Save',
    color: 'primary',
    onClick: doSave
  }));
  const refreshNeeded = computed(() => remoteIsChanged.value);
  const refreshAttrs = computed(() => ({
    text: 'Refresh',
    color: 'warning',
    onClick: sync
  }));

  const fabLoadingAttrs = {disabled: true, loading: true, icon: 'mdi-content-save'};
  const fabDisabledAttrs = {disabled: true, icon: 'mdi-content-save'};
  const fabActiveAttrs = computed(() => ({
    prependIcon: 'mdi-content-save',
    text: 'Save',
    extended: true,
    rounded: true,
    size: 'large',
    color: 'primary',
    readonly: saveTracker.loading,
    loading: saveTracker.loading,
    onClick: doSave
  }));
  const fabRefreshAttrs = computed(() => ({
    prependIcon: 'mdi-refresh',
    text: 'Refresh',
    extended: true,
    rounded: true,
    size: 'large',
    color: 'warning',
    onClick: sync
  }));
  const fabErrorAttrs = computed(() => ({
    prependIcon: 'mdi-alert',
    text: 'Error',
    extended: true,
    rounded: true,
    size: 'large',
    color: 'error',
    readonly: saveTracker.loading,
    loading: saveTracker.loading,
    onClick: doSave
  }));

  const fabAttrs = computed(() => {
    if (blockActions.value) return fabDisabledAttrs;
    if (localParseError.value) return fabDisabledAttrs;
    if (!configModel.value) return fabLoadingAttrs.value;
    if (saveTracker.error) return fabErrorAttrs.value;
    if (remoteIsChanged.value) return fabRefreshAttrs.value;
    if (localIsChanged.value) return fabActiveAttrs.value;
    return fabDisabledAttrs;
  });

  const refreshTooltip = computed(() => {
    return 'External changes detected, refresh to fetch the latest information';
  });
  const saveTooltip = computed(() => {
    if (blockActions.value) return 'You do not have permission to change the configuration';
    if (localParseError.value) return 'Invalid JSON: ' + localParseError.value.message;
    if (!configModel.value) return 'Loading...';
    if (saveTracker.error) return 'Error saving changes, click to try again.\n' + formatErrorMessage(saveTracker.error.error.message);
    if (remoteIsChanged.value) return refreshTooltip.value;
    if (localIsChanged.value) return 'Save changes';
    return 'No changes have been made';
  });

  const alertVisible = computed(() => !!serviceError.value);
  const alertAttrs = computed(() => {
    const attrs = {
      type: 'error'
    };
    if (saveTracker.error) attrs.text = errorString(saveTracker.error, 'Error saving service configuration');
    else if (serviceError.value) attrs.text = errorString(serviceError.value, 'Error fetching service configuration');
    return attrs;
  });

  const serviceID = computed(() => service.value?.id);
  const serviceType = computed(() => service.value?.type);
  const serviceDisabled = computed(() => !!service.value?.disabled);

  return {
    // for when the save and refresh buttons are combined
    fabAttrs,
    // for when the save and refresh buttons are separate
    saveAttrs,
    refreshNeeded,
    refreshAttrs,
    // tooltips should be applied separately to a containing element
    refreshTooltip,
    saveTooltip,

    alertVisible,
    alertAttrs,

    loading: serviceLoading,
    remoteModel: configParsed,
    configModel: localChanges,
    readonly: blockActions,

    serviceID,
    serviceType,
    serviceDisabled
  };
}

const errorString = (error, prefix = 'An error occurred') => {
  if (!error) return '';
  if (error.error) error = error.error;
  const parts = [prefix + ':'];
  if (error.code) parts.push(statusCodeToString(error.code));
  if (error.message) parts.push(formatErrorMessage(error.message));
  return parts.join(' ');
};
