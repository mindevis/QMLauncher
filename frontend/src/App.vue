<script setup lang="ts">
import HelloWorld from './components/HelloWorld.vue'
import Button from './components/ui/button.vue'
import Card from './components/ui/card.vue'
import CardHeader from './components/ui/card-header.vue'
import CardTitle from './components/ui/card-title.vue'
import CardContent from './components/ui/card-content.vue'
</script>

<template>
  <div class="min-h-screen bg-background p-8">
    <div class="max-w-4xl mx-auto space-y-8">
      <!-- Header -->
      <div class="text-center space-y-4">
        <img
          id="logo"
          alt="QMLauncher logo"
          src="./assets/images/logo-universal.png"
          class="w-32 h-32 mx-auto"
        />
        <h1 class="text-4xl font-bold text-foreground">QMLauncher</h1>
        <p class="text-lg text-muted-foreground">
          Modern desktop application built with Wails, Go and Vue.js
        </p>
      </div>

      <!-- Main content -->
      <div class="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Welcome to QMLauncher</CardTitle>
          </CardHeader>
          <CardContent>
            <p class="text-muted-foreground mb-4">
              This is a demonstration of shadcn-vue components integrated with Wails.
            </p>
            <div class="space-y-2">
              <Button variant="default" class="w-full">
                Get Started
              </Button>
              <Button variant="outline" class="w-full">
                Learn More
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Features</CardTitle>
          </CardHeader>
          <CardContent>
            <ul class="space-y-2 text-sm text-muted-foreground">
              <li>✅ Cross-platform desktop application</li>
              <li>✅ Modern Vue.js frontend</li>
              <li>✅ Go backend with Wails</li>
              <li>✅ shadcn-vue UI components</li>
              <li>✅ Tailwind CSS styling</li>
              <li>✅ Hot reload development</li>
            </ul>
          </CardContent>
        </Card>
      </div>

      <!-- Legacy component for compatibility -->
      <HelloWorld />
    </div>
  </div>
</template>
