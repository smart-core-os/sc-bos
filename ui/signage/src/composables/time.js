import {isNullOrUndef} from '@/util/types.js';
import {
  add,
  addDays,
  addHours,
  addMinutes,
  addMonths,
  addWeeks,
  addYears,
  startOfDay,
  startOfHour,
  startOfMinute,
  startOfMonth,
  startOfWeek,
  startOfYear, toDate
} from 'date-fns';
import {computed, effectScope, onMounted, onScopeDispose, onUnmounted, reactive, ref, toValue, watch} from 'vue';
import {setTimeout, clearTimeout} from 'safe-timers';

/**
 * Returns values that can show time since a given time.
 *
 * @example
 * const {showTimeSince, timeSinceStr} = useTimeSince(updateDate);
 * const tooltipStr = computed(() => {
 *   if (showTimeSince.value) {
 *     return `${stateStr.value} for ${timeSinceStr.value}`;
 *   } else {
 *     return stateStr.value;
 *   }
 * })
 * // tooltipStr output will be something like "Active for 5m"
 *
 * @param {Date} date
 * @return {{
 *   now: Ref<Date>,
 *   showTimeSince: ComputedRef<boolean>,
 *   timeSinceStr: ComputedRef<string>
 * }}
 */
export function useTimeSince(date) {
  const nowHandle = ref(0);
  const now = ref(new Date());
  onUnmounted(() => clearInterval(nowHandle.value));
  onMounted(() => {
    nowHandle.value = setInterval(() => {
      now.value = new Date();
    }, 1000);
  });

  const millisSince = computed(() => {
    return now.value.getTime() - toValue(date)?.getTime();
  });
  const showTimeSince = computed(() => {
    return Boolean(toValue(date) && millisSince.value > 1000);
  });
  const timeSinceStr = computed(() => {
    if (!showTimeSince.value) return '';
    const t = millisSince.value;
    if (t > 1000 * 60 * 60 * 24) {
      const h = Math.floor(t / (1000 * 60 * 60 * 24));
      return `${h}d`;
    } else if (t > 1000 * 60 * 60) {
      const h = Math.floor(t / (1000 * 60 * 60));
      return `${h}h`;
    } else if (t > 1000 * 60) {
      const m = Math.floor(t / (1000 * 60));
      return `${m}m`;
    } else if (t > 1000) {
      const s = Math.floor(t / 1000);
      return `${s}s`;
    } else {
      return '';
    }
  });

  return {
    now,
    showTimeSince,
    timeSinceStr
  };
}

/**
 * Returns a ref to a date that is updated to represent the start of a period.
 * The startOf function converts a date to the start of a period.
 * The nextOf function converts a date to an equivalent date in the next period.
 *
 * @example
 * const t = useStartOfPeriod(startOfDay, addDays);
 * // t will always be the start of the current day
 *
 * @param {function(Date):Date} startOf - return a new date that is at the start of the period enclosing the passed Date
 * @param {function(Date, number):Date} nextOf - return a new date that is some number of periods after the passed Date
 * @return {import('vue').Ref<Date>}
 */
export function useStartOfPeriod(startOf, nextOf) {
  const tRef = ref(startOf(new Date()));
  let startTimeHandle = 0;
  const updateStartTime = () => {
    const n = new Date();
    const t = startOf(n);
    tRef.value = t;
    const startOfNext = startOf(nextOf(t, 1));
    startTimeHandle = setTimeout(updateStartTime, startOfNext.getTime() - n.getTime());
  };

  updateStartTime();
  onScopeDispose(() => {
    clearTimeout(startTimeHandle);
  });

  return tRef;
}

export const useStartOf = {
  minute: () => useStartOfPeriod(startOfMinute, addMinutes),
  hour: () => useStartOfPeriod(startOfHour, addHours),
  day: () => useStartOfPeriod(startOfDay, addDays),
  week: () => useStartOfPeriod(startOfWeek, addWeeks),
  month: () => useStartOfPeriod(startOfMonth, addMonths),
  year: () => useStartOfPeriod(startOfYear, addYears)
};

/**
 * @typedef {keyof useStartOf} NamedPeriod
 */

/**
 * Returns whether the given period is a known named period.
 * Named periods can be used with functions like usePeriod and exist in useStartOf.
 *
 * @param {any} period
 * @return {boolean}
 */
export function isNamedPeriod(period) {
  return typeof period === 'string' && Object.hasOwn(useStartOf, period);
}

/**
 * @type {Record<keyof useStartOf, keyof useStartOf>}
 */
export const previousNamedPeriod = {
  hour: 'minute',
  day: 'hour',
  week: 'day',
  month: 'day',
  year: 'month'
};

/**
 * Tracks a period of time.
 * The start and end dates can be named periods like 'day' or 'week', in which case the period will represent the current period,
 * i.e. between the start and end of this week.
 * Named dates can have individual offsets using a format like 'day - 7',
 * for example using start='day-30', end='day' will represent a period spanning the last 30 days including today.
 *
 * When using named dates, a universal offset can be used to adjust the period in use.
 * For example an offset of -1 with a period of 'day' will represent the previous day.
 *
 * @param {import('vue').MaybeRefOrGetter<keyof useStartOf | string | number | Date>} start
 * @param {import('vue').MaybeRefOrGetter<keyof useStartOf | string | number | Date>} end
 * @param {import('vue').MaybeRefOrGetter<null | undefined | number | string>} [offset]
 * @return {{
 *   start: import('vue').Ref<Date | null>,
 *   end: import('vue').Ref<Date | null>
 * }}
 */
export function usePeriod(start, end, offset) {
  const _offset = computed(() => {
    const o = toValue(offset);
    if (!o) return 0;
    return parseInt(o);
  });
  /**
   * @param {string} period
   * @param {import('vue').MaybeRefOrGetter<Date>} t
   * @param {number} [plus]
   * @return {ComputedRef<Date>}
   */
  const useOffset = (period, t, plus = 0) => {
    return computed(() => {
      const o = _offset.value + plus;
      if (o === 0) return toValue(t);
      return add(toValue(t), {[`${period}s`]: o});
    });
  };

  // matches strings like 'day - 7', 'week+1.5'
  const boundRe = /^(?<period>[a-z]+)(?:\s*(?<offset>[-+]\s*[\d.]+))?$/;
  /**
   * A helper composable that returns a ref to a date at the start of t + plus.
   *
   * @param {import('vue').MaybeRefOrGetter<NamedPeriod | Date | number | string>} t
   * @param {number} [plus]
   * @return {import('vue').ComputedRef<null | Date>}
   */
  const useStartOfBound = (t, plus = 0) => {
    const res = reactive({value: /** @type {Date | null} */ null});

    let closeScope = () => {};
    onScopeDispose(() => closeScope());

    const parsedT = computed(() => {
      const _t = toValue(t);
      if (typeof _t === 'number') return {t: _t, o: 0};
      if (typeof _t !== 'string') return {t: _t, o: 0};
      if (_t instanceof Date) return {t: _t, o: 0};
      const matches = boundRe.exec(_t);
      if (!matches) return {t: _t, o: 0}; // assume it's a date string
      return {t: matches.groups.period, o: parseFloat(matches.groups.offset) || 0};
    });

    watch(parsedT, (t, oldT) => {
      if (oldT === t) return;
      closeScope();
      if (isNullOrUndef(t)) {
        res.value = null;
        return;
      }
      if (isNamedPeriod(t.t)) {
        const scope = effectScope();
        closeScope = () => scope.stop();
        scope.run(() => {
          const d = useStartOf[t.t]();
          res.value = useOffset(t.t, d, plus+t.o);
        });
        return;
      }

      // There's a quirk of Vue that we have to work around here.
      // If res.value had ever been a ref, i.e. t used to be a named period,
      // then when we reassign res.value here Vue will check that whether that
      // assignment should be written to the current ref value.
      // As our named period branch assigns it to a computed prop, Vue will
      // fail at that step because computed props are read-only.
      //
      // To get around this we assign a ref to res.value which causes Vue to
      // replace res.value with the new ref instead of trying to write to the old ref.
      res.value = computed(() => {
        if (typeof t.t === 'number') return new Date(t.t);
        if (typeof t.t === 'string') return new Date(t.t);
        if (t.t instanceof Date) return t.t;
        return null;
      });
    }, {immediate: true});

    return computed(() => res.value);
  };

  return {
    start: useStartOfBound(start),
    end: useStartOfBound(end, 1)
  };
}

/**
 * Returns a ref containing the leading segment of dates that are in the past.
 *
 * @example
 * // if now is 14:10
 * const dates = usePastDates([13:00, 14:00, 15:00, 16:00]);
 * // dates.value will be [13:00, 14:00]
 *
 * @param {import('vue').MaybeRefOrGetter<Date[]>} dates
 * @return {import('vue').ComputedRef<Date[]>}
 */
export function usePastDates(dates) {
  const now = ref(new Date()); // the current time, but in a resolution suitable for working out the tense of dates
  let nowHandle = 0;
  onScopeDispose(() => clearTimeout(nowHandle));

  watch(() => toValue(dates), (dates) => {
    const updateNow = () => {
      clearTimeout(nowHandle);
      const t = new Date();
      now.value = t;
      const nextDate = dates.find((b) => b.getTime() > t.getTime());
      if (!nextDate) return;
      const delay = nextDate.getTime() - t.getTime();
      nowHandle = setTimeout(() => updateNow(), delay);
    };
    updateNow();
  }, {immediate: true});

  return computed(() => {
    const t = now.value.getTime();
    const _dates = toValue(dates);
    if (_dates[_dates.length - 1].getTime() < t) return _dates;
    const res = [];
    for (let i = 0; i < _dates.length; i++) {
      if (_dates[i].getTime() > t) break;
      res.push(_dates[i]);
    }
    return res;
  });
}

/**
 * Returns a ref that says whether the given date is in the future.
 * The ref will update as time changes.
 *
 * @param {import('vue').MaybeRefOrGetter<Date | number>} date
 * @return {import('vue').ComputedRef<boolean>}
 */
export function useIsFutureDate(date) {
  const _date = computed(() => toDate(toValue(date)));
  const now = ref(new Date());
  let nowHandle = 0;
  onScopeDispose(() => clearTimeout(nowHandle));

  watch(_date, (date) => {
    const updateNow = () => {
      clearTimeout(nowHandle);
      const t = new Date();
      now.value = t;
      const delay = date.getTime() - t.getTime();
      nowHandle = setTimeout(() => updateNow(), delay);
    };
    updateNow();
  }, {immediate: true});

  return computed(() => {
    return now.value.getTime() < _date.value.getTime();
  });
}
