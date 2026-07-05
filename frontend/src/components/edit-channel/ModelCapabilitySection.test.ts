// @vitest-environment jsdom
/* eslint-disable vue/one-component-per-file, vue/require-prop-types */
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import ModelCapabilitySection from './ModelCapabilitySection.vue'

vi.mock('../../i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const passthroughStub = defineComponent({
  template: '<div><slot /></div>',
})

const tooltipStub = defineComponent({
  template: '<div><slot name="activator" :props="{}" /><slot /></div>',
})

const comboboxStub = defineComponent({
  name: 'VCombobox',
  props: [
    'density',
    'eager',
    'hideDetails',
    'itemTitle',
    'itemValue',
    'items',
    'label',
    'loading',
    'menuProps',
    'modelValue',
    'noDataText',
    'placeholder',
    'variant',
  ],
  template: '<div />',
})

const vuetifyStubs = {
  VBtn: passthroughStub,
  VCard: passthroughStub,
  VCardText: passthroughStub,
  VCardTitle: passthroughStub,
  VChip: passthroughStub,
  VCol: passthroughStub,
  VCombobox: comboboxStub,
  VIcon: passthroughStub,
  VRow: passthroughStub,
  VTextField: passthroughStub,
  VTooltip: tooltipStub,
}

describe('ModelCapabilitySection', () => {
  it('将新增实际模型菜单挂载到 body，避免被后续配置块遮挡', () => {
    const wrapper = mount(ModelCapabilitySection, {
      props: {
        rows: [],
        targetModelOptions: [{ title: 'claude-opus-4-8', value: 'claude-opus-4-8' }],
        mappedTargetModels: [],
        fetchingModels: false,
        fetchModelsError: '',
        error: '',
      },
      global: {
        stubs: vuetifyStubs,
      },
    })

    const modelCombobox = wrapper
      .findAllComponents({ name: 'VCombobox' })
      .find(combobox => combobox.props('label') === 'addChannel.modelCapabilityModelLabel')

    expect(modelCombobox).toBeTruthy()
    expect(modelCombobox?.props('menuProps')).toMatchObject({
      attach: 'body',
      zIndex: expect.any(Number),
    })
    expect(modelCombobox?.props('menuProps').zIndex).toBeGreaterThan(2000)
  })
})
