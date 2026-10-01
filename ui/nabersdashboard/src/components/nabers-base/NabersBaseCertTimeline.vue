<template>
  <div class="cert-wrapper">
    <div class="cert-title">Certification timeline</div>

    <div v-if="assessmentType" class="cert-row">
      <span class="cert-label">Assessment</span>
      <v-chip color="#7F00FF" size="small" variant="tonal">{{ assessmentType }}</v-chip>
    </div>

    <div class="cert-row">
      <span class="cert-label">Certificate</span>
      <span class="cert-value" :style="{color: certColor}">{{ certStatus }}</span>
    </div>

    <div v-if="certExpiryLabel" class="cert-row">
      <span class="cert-label">Expires</span>
      <span class="cert-value">{{ certExpiryLabel }}</span>
    </div>

    <div v-if="submissionDeadlineLabel" class="cert-row">
      <span class="cert-label">Next submission</span>
      <span class="cert-value" :style="{color: submissionColor}">{{ submissionDeadlineLabel }}</span>
    </div>

    <div class="cert-row">
      <span class="cert-label">Occupancy cert.</span>
      <span class="cert-value">{{ occupancyCertificateLabel }}</span>
    </div>

    <div class="cert-desc">
      Certificate valid for 12 months. Annual resubmission to accredited assessor required. Rating adjusted for actual occupancy hours and leased area in operation.
    </div>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {addMonths, subWeeks, format, parseISO, isValid, differenceInDays} from 'date-fns';

const props = defineProps({
  certIssueDate:            {type: String, default: ''},
  occupancyCertificateDate: {type: String, default: ''},
  /** e.g. "Design-Reviewed Target Rating" or "In-use" — from the assessment. */
  assessmentType:           {type: String, default: ''}
});

const certExpiry = computed(() => {
  if (!props.certIssueDate) return null;
  try {
    const issued = parseISO(props.certIssueDate);
    return isValid(issued) ? addMonths(issued, 12) : null;
  } catch { return null; }
});

const daysToExpiry = computed(() =>
  certExpiry.value ? differenceInDays(certExpiry.value, new Date()) : null
);

const certStatus = computed(() => {
  if (!certExpiry.value) return 'Not yet certified';
  if (daysToExpiry.value < 0) return 'Expired';
  if (daysToExpiry.value <= 28) return `Expires in ${daysToExpiry.value} days`;
  return 'Active';
});

const certColor = computed(() => {
  if (!certExpiry.value) return 'rgba(255,255,255,0.4)';
  if (daysToExpiry.value < 0) return '#f89c9b';
  if (daysToExpiry.value <= 28) return '#f7a12c';
  return '#4caf50';
});

const certExpiryLabel = computed(() =>
  certExpiry.value ? format(certExpiry.value, 'd MMM yyyy') : null
);

const submissionDeadline = computed(() =>
  certExpiry.value ? subWeeks(certExpiry.value, 4) : null
);

const submissionDeadlineLabel = computed(() =>
  submissionDeadline.value ? format(submissionDeadline.value, 'd MMM yyyy') : null
);

const submissionColor = computed(() => {
  if (!submissionDeadline.value) return 'rgba(255,255,255,0.6)';
  const days = differenceInDays(submissionDeadline.value, new Date());
  if (days < 0) return '#f89c9b';
  if (days <= 14) return '#f7a12c';
  return 'rgba(255,255,255,0.6)';
});

const occupancyCertificateLabel = computed(() => {
  if (!props.occupancyCertificateDate) return 'Not set';
  try {
    const d = parseISO(props.occupancyCertificateDate);
    return isValid(d) ? format(d, 'd MMM yyyy') : 'Not set';
  } catch { return 'Not set'; }
});
</script>

<style scoped>
.cert-wrapper {
  background:     rgba(255, 255, 255, 0.05);
  border:         1px solid rgba(255, 255, 255, 0.08);
  border-radius:  12px;
  padding:        20px 24px;
  display:        flex;
  flex-direction: column;
  gap:            14px;
  flex:           1;
}

.cert-title {
  font-size:      11px;
  font-weight:    500;
  color:          rgba(255, 255, 255, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.cert-row {
  display:     flex;
  align-items: center;
  gap:         12px;
}

.cert-label {
  font-size:  13px;
  color:      rgba(255, 255, 255, 0.4);
  min-width:  130px;
  flex-shrink: 0;
}

.cert-value {
  font-size:   16px;
  font-weight: 400;
  color:       #ffffff;
}

.cert-desc {
  font-size:   13px;
  color:       rgba(255, 255, 255, 0.35);
  margin-top:  auto;
  line-height: 1.4;
}
</style>
