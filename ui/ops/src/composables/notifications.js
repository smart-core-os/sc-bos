import {acknowledgeAlert, unacknowledgeAlert} from '@/api/ui/alerts.js';
import {useAccountStore} from '@/stores/account.js';
import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alert/v1/alerts_pb';

export const SeverityStrings = {
  [Alert.Severity.INFO]: 'INFO',
  [Alert.Severity.WARNING]: 'WARN',
  [Alert.Severity.SEVERE]: 'ALERT',
  [Alert.Severity.LIFE_SAFETY]: 'DANGER'
};
export const SeverityColor = {
  [Alert.Severity.INFO]: 'info',
  [Alert.Severity.WARNING]: 'warning',
  [Alert.Severity.SEVERE]: 'error',
  [Alert.Severity.LIFE_SAFETY]: 'error'
};

/**
 *
 * @param {Severity} severity
 * @return {{text: string, color: string, background: string}}
 */
export function severityData(severity) {
  for (let i = severity; i > 0; i--) {
    if (SeverityStrings[i]) {
      let str = SeverityStrings[i];
      if (i < severity) {
        str += '+' + (severity - i);
      }
      return {text: str, color: `text-${SeverityColor[i]}`, background: `bg-${SeverityColor[i]}`};
    }
  }
  return {text: 'unspecified', color: 'text-gray', background: 'gray'};
}

export const useAcknowledgement = () => {
  const account = useAccountStore();

  /**
   *
   * @param {boolean} e - true to acknowledge, false to unacknowledge
   * @param {Alert.AsObject} alert - the alert to acknowledge or unacknowledge, must have an id
   * @param {string} name - the device that holds the alert data
   */
  function setAcknowledged(e, alert, name = '') {
    if (e) {
      let author = undefined;
      if (account.email || account.fullName) {
        author = {
          email: account.email,
          displayName: account.fullName
        };
      }
      acknowledgeAlert({
        name,
        id: alert.id,
        allowAcknowledged: false,
        allowMissing: false,
        author
      }).catch((err) => console.error(err));
    } else {
      unacknowledgeAlert({name, id: alert.id, allowAcknowledged: false, allowMissing: false})
          .catch((err) => console.error(err));
    }
  }

  return {
    setAcknowledged
  };
};
