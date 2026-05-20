<script setup lang="ts">
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

const activeTab = defineModel<string>({ default: 'status' })
</script>

<template>
  <Tabs v-model="activeTab" class="w-full">
    <TabsList class="inline-flex h-9 items-center rounded-full bg-secondary p-1 gap-1">
      <TabsTrigger
        value="status"
        class="rounded-full px-4 text-sm data-[state=active]:bg-primary data-[state=active]:text-primary-foreground transition-all"
      >
        服务状态
      </TabsTrigger>
      <TabsTrigger
        value="agent"
        class="rounded-full px-4 text-sm data-[state=active]:bg-primary data-[state=active]:text-primary-foreground transition-all"
      >
        Agent 配置
      </TabsTrigger>
      <TabsTrigger
        value="env"
        class="rounded-full px-4 text-sm data-[state=active]:bg-primary data-[state=active]:text-primary-foreground transition-all"
      >
        环境配置
      </TabsTrigger>
      <TabsTrigger
        value="web"
        class="rounded-full px-4 text-sm data-[state=active]:bg-primary data-[state=active]:text-primary-foreground transition-all"
      >
        Web UI
      </TabsTrigger>
    </TabsList>

    <TabsContent value="status" class="mt-4 focus-visible:outline-none">
      <slot name="status" />
    </TabsContent>
    <TabsContent value="agent" class="mt-4 focus-visible:outline-none">
      <slot name="agent" />
    </TabsContent>
    <TabsContent value="env" class="mt-4 focus-visible:outline-none">
      <slot name="env" />
    </TabsContent>
    <TabsContent value="web" class="mt-4 focus-visible:outline-none">
      <slot name="web" />
    </TabsContent>
  </Tabs>
</template>
