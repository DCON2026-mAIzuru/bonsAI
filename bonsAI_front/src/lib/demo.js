const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

const demoSensorFrames = [
  { temperature: 24.6, humidity: 58, soilMoisture: 43, illuminance: 12800 },
  { temperature: 24.9, humidity: 57, soilMoisture: 41, illuminance: 13200 },
  { temperature: 25.1, humidity: 55, soilMoisture: 38, illuminance: 14000 },
  { temperature: 24.7, humidity: 60, soilMoisture: 36, illuminance: 11800 },
  { temperature: 24.3, humidity: 62, soilMoisture: 44, illuminance: 9600 },
  { temperature: 24.1, humidity: 59, soilMoisture: 47, illuminance: 8800 }
];

function resolveDemoSlot(date = new Date()) {
  return Math.floor(date.getTime() / 10000);
}

export function createDemoSensorsSnapshot(date = new Date()) {
  const slot = resolveDemoSlot(date);
  const values = demoSensorFrames[slot % demoSensorFrames.length];

  return {
    ...values,
    lastUpdated: new Date(slot * 10000).toISOString(),
    source: "fallback"
  };
}

export const demoSensors = createDemoSensorsSnapshot();

export function createVeryDryDemoSensorsSnapshot(date = new Date()) {
  return {
    temperature: 27.2,
    humidity: 34,
    soilMoisture: 12,
    illuminance: 15400,
    lastUpdated: date.toISOString(),
    source: "demo-very-dry"
  };
}

export async function streamDemoReply({ message, sensors, onDelta }) {
  const lowerMessage = message.toLowerCase();
  const needsWater =
    typeof sensors.soilMoisture === "number" && sensors.soilMoisture < 35;
  const brightEnough =
    typeof sensors.illuminance === "number" && sensors.illuminance > 9000;

  let response =
    "こんにちは。盆栽の今の気配を見ながら、一緒に様子を整理してみます。";

  if (lowerMessage.includes("水") || lowerMessage.includes("乾")) {
    response = needsWater
      ? "土壌水分が少し低めです。今日は少量ずつ様子を見ながら水やりを検討してよさそうです。"
      : "土壌水分はまだ保たれています。今すぐ急いで水を足すより、表土の乾き方を見ながらで大丈夫そうです。";
  } else if (lowerMessage.includes("元気") || lowerMessage.includes("状態")) {
    response = brightEnough
      ? "照度は確保できていて、盆栽は比較的落ち着いた環境にいます。温湿度と土壌水分のバランスも大きく崩れていません。"
      : "強い異常は見えませんが、やや光量が控えめです。置き場所の明るさを少しだけ見直すと変化が分かりやすいかもしれません。";
  } else if (lowerMessage.includes("ありがとう")) {
    response =
      "どういたしまして。小さな変化を一緒に見ていけるのが、この盆栽UIのいちばん良いところです。";
  } else {
    response = `${response} 現在は${sensors.temperature}℃、湿度${sensors.humidity}%、土壌水分${sensors.soilMoisture}%、照度${sensors.illuminance}lxです。質問があれば、水やり・日照・体調の見立てに寄せて返答できます。`;
  }

  const chunks = response.split(/(?<=[。 ]|、)/).filter(Boolean);

  for (const chunk of chunks) {
    await wait(130);
    onDelta(chunk);
  }
}
