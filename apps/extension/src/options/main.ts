import { createApp, defineComponent } from "vue";
import "../styles.css";

const OptionsApp = defineComponent({
  template: `
    <main class="options-shell">
      <section class="card">
        <h1>TradeMate Extension Settings</h1>
        <p>This page is a placeholder for reminder, display, and store preferences.</p>
      </section>
    </main>
  `
});

createApp(OptionsApp).mount("#app");

