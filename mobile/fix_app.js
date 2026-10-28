const fs = require('fs');
const file = 'C:\\Users\\USRE\\Downloads\\taxi-connect\\mobile\\App.tsx';
let c = fs.readFileSync(file, 'utf8');
let changed = 0;

// 1. Remove unused ScrollView from import
{
  const token = 'ScrollView,';
  const idx = c.indexOf(token);
  if (idx === -1) throw new Error('ScrollView not found in import');
  const lineStart = c.lastIndexOf('\n', idx) + 1;
  const lineEnd = c.indexOf('\n', idx) + 1;
  c = c.slice(0, lineStart) + c.slice(lineEnd);
  changed++;
  console.log('[1] Removed ScrollView import');
}

// 2. Remove standalone terminal selector block (non-rendering, triggers no-unused-expressions)
{
  const comment = '{/* Select an origin terminal from the nearby list */}';
  const start = c.indexOf(comment);
  if (start === -1) throw new Error('selector comment not found');
  const endMarker = c.indexOf('  </View>', start);
  if (endMarker === -1) throw new Error('selector close View not found');
  const end = endMarker + '  </View>'.length;
  // consume trailing blank lines up to the next statement
  let after = end;
  while (after < c.length && /\s/.test(c[after])) after++;
  c = c.slice(0, start) + c.slice(after);
  changed++;
  console.log('[2] Removed standalone terminal selector block');
}

// 3. Add handleSelectTerminal that uses both setSelectedOrigin and setSelectedDestination
{
  const token = 'const [routeInfo, setRouteInfo] = useState<RouteInfo | null>(null);';
  const idx = c.indexOf(token);
  if (idx === -1) throw new Error('routeInfo state not found');
  const insertPos = idx + token.length;
  const handler = `

const handleSelectTerminal = (id: string) => {
  if (!selectedOrigin) {
    setSelectedOrigin(id);
    setSelectedDestination(null);
    setRouteInfo(null);
  } else if (!selectedDestination) {
    setSelectedDestination(id);
  } else {
    setSelectedOrigin(id);
    setSelectedDestination(null);
    setRouteInfo(null);
  }
};`;
  c = c.slice(0, insertPos) + handler + c.slice(insertPos);
  changed++;
  console.log('[3] Added handleSelectTerminal');
}

// 4. Restructure route-fetch effect: remove synchronous setRouteInfo(null) from effect body
{
  const old = 'if (!selectedOrigin || !selectedDestination) {\n      setRouteInfo(null);\n      return;';
  const newStr = 'if (!selectedOrigin || !selectedDestination) {\n      return;';
  const idx = c.indexOf(old);
  if (idx === -1) throw new Error('effect early-return not found');
  c = c.slice(0, idx) + newStr + c.slice(idx + old.length);
  changed++;
  console.log('[4] Removed synchronous setRouteInfo(null) from effect body');
}

// 5. Insert terminal selector JSX into the return (before route selection panel)
{
  const anchor = '      {/* Route selection: chosen origin and destination */}';
  const idx = c.indexOf(anchor);
  if (idx === -1) throw new Error('route selection anchor not found');
  const selector = '      {/* Terminal selector: pick an origin and destination */}\n      <View style={styles.terminalSelector}>\n        {terminals.map((terminal) => (\n          <Pressable\n            key={terminal.id}\n            style={styles.selectorItem}\n            onPress={() => handleSelectTerminal(terminal.id)}\n          >\n            <Text>{terminal.name}</Text>\n          </Pressable>\n        ))}\n      </View>\n\n';
  c = c.slice(0, idx) + selector + c.slice(idx);
  changed++;
  console.log('[5] Inserted terminal selector into return JSX');
}

fs.writeFileSync(file, c, 'utf8');
console.log('Done. Total changes:', changed);
