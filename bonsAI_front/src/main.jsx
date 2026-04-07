import { render } from "preact";
import App from "./app.jsx";
import "./styles.css";
import { registerPwa } from "./lib/pwa.js";

registerPwa();
render(<App />, document.getElementById("app"));
