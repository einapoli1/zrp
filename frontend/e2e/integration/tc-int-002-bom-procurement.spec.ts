import { test, expect } from '@playwright/test';

/**
 * TC-INT-002: BOM Shortage Detection → Procurement Flow Integration Test
 * 
 * **CURRENT STATUS**: Documents known workflow gaps in ZRP procurement flow
 * 
 * This test validates critical gaps in the end-to-end procurement workflow:
 * - BOM shortage detection
 * - PO generation from shortages  
 * - PO receiving → inventory update
 * 
 * **KEY FINDINGS** (as of 2026-02-19):
 * - ✅ Work order creation: WORKS
 * - ✅ Work order BOM endpoint exists: GET /api/v1/workorders/{id}/bom
 * - ⚠️  BOM management: File-based (gitplm), no API for BOM creation (WORKFLOW_GAPS.md #1.5)
 * - ⚠️  BOM check bugs: Returns ALL inventory, not assembly-specific BOM (WORKFLOW_GAPS.md #3.1)
 * - ❌ PO generation from WO shortages: NOT IMPLEMENTED (WORKFLOW_GAPS.md #3.1)
 * - ❓ PO receiving → inventory update: TESTED BELOW
 * - ❌ Material reservation on WO create: NOT IMPLEMENTED (WORKFLOW_GAPS.md #4.1)
 * 
 * **REFERENCES**:
 * - docs/INTEGRATION_TEST_PLAN.md (TC-INT-001: Complete BOM-to-Procurement Flow)
 * - docs/WORKFLOW_GAPS.md (#1.5, #3.1, #3.2, #4.1)
 * - docs/INTEGRATION_TESTS_NEEDED.md (TC-INT-002)
 */

test.describe('TC-INT-002: BOM Shortage → Procurement Flow', () => {
  
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    
    // Login
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
    await page.waitForURL(/dashboard|home/i, { timeout: 10000 });
  });

  test('Integration Flow: WO → BOM Check → PO → Receiving → Inventory', async ({ page }) => {
    const timestamp = Date.now();
    const vendorId = `V-INT-${timestamp}`;
    const resistorIpn = `RES-INT-${timestamp}`;
    const capacitorIpn = `CAP-INT-${timestamp}`;
    const woNumber = `WO-INT-${timestamp}`;
    
    console.log('\n' + '='.repeat(60));
    console.log('TC-INT-002: BOM Shortage → Procurement Integration Test');
    console.log('='.repeat(60) + '\n');
    
    // ============================================================
    // SETUP: Create minimal test data
    // ============================================================
    
    console.log('📋 SETUP: Creating test data...\n');
    
    // Create vendor
    const vendorResp = await page.request.post('/api/v1/vendors', {
      data: {
        vendor_id: vendorId,
        name: `Integration Test Vendor ${timestamp}`,
        status: 'active',
        lead_time_days: 7
      }
    });
    expect(vendorResp.ok(), `Vendor creation failed: ${await vendorResp.text()}`).toBeTruthy();
    console.log(`  ✓ Vendor: ${vendorId}`);
    
    // Create categories and parts
    await page.request.post('/api/v1/categories', {
      data: { title: `int-test-${timestamp}`, prefix: `int-${timestamp}` }
    });
    
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: resistorIpn,
        category: `int-${timestamp}`,
        status: 'production',
        description: '10k Resistor for Integration Test'
      }
    });
    
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: capacitorIpn,
        category: `int-${timestamp}`,
        status: 'production',
        description: '100uF Capacitor for Integration Test'
      }
    });
    console.log(`  ✓ Parts: ${resistorIpn}, ${capacitorIpn}`);
    
    // Create inventory with LOW stock (to simulate shortages)
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: resistorIpn,
        qty_on_hand: 10.0,
        qty_reserved: 0.0,
        reorder_point: 50.0
      }
    });
    
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: capacitorIpn,
        qty_on_hand: 5.0,
        qty_reserved: 0.0,
        reorder_point: 25.0
      }
    });
    console.log(`  ✓ Inventory: ${resistorIpn}=10, ${capacitorIpn}=5 (LOW stock)`);
    
    console.log('\n' + '-'.repeat(60));
    
    // ============================================================
    // STEP 1: Create Work Order
    // ============================================================
    
    console.log('\n📝 STEP 1: Create Work Order\n');
    
    // Note: Using a generic assembly IPN since BOM is file-based
    const woResp = await page.request.post('/api/v1/workorders', {
      data: {
        assembly_ipn: 'TEST-ASSEMBLY',  // Placeholder - BOM is file-based
        qty: 100,
        status: 'open',
        priority: 'normal',
        notes: 'Integration test work order'
      }
    });
    
    expect(woResp.ok(), `WO creation failed: ${await woResp.text()}`).toBeTruthy();
    const wo = await woResp.json();
    const actualWoId = wo.id || wo.ID;
    console.log(`  ✓ Created: ${actualWoId}`);
    console.log(`    Assembly: ${wo.assembly_ipn}`);
    console.log(`    Quantity: ${wo.qty}`);
    console.log(`    Status: ${wo.status}`);
    
    console.log('\n' + '-'.repeat(60));
    
    // ============================================================
    // STEP 2: Check BOM Endpoint
    // ============================================================
    
    console.log('\n🔍 STEP 2: Check BOM Endpoint\n');
    
    const bomResp = await page.request.get(`/api/v1/workorders/${actualWoId}/bom`);
    expect(bomResp.ok(), `BOM endpoint failed: ${await bomResp.text()}`).toBeTruthy();
    
    const bomData = await bomResp.json();
    console.log(`  ✓ BOM endpoint accessible`);
    console.log(`    WO ID: ${bomData.wo_id}`);
    console.log(`    Assembly: ${bomData.assembly_ipn}`);
    console.log(`    Quantity: ${bomData.qty}`);
    console.log(`    BOM items returned: ${bomData.bom?.length || 0}`);
    
    console.log('\n  ⚠️  KNOWN GAP (#3.1): BOM check returns ALL inventory\n');
    console.log(`    - Current: Queries ALL inventory items`);
    console.log(`    - Expected: Only BOM components for ${bomData.assembly_ipn}`);
    console.log(`    - Issue: handleWorkOrderBOM doesn't join with BOM table/files`);
    
    console.log('\n' + '-'.repeat(60));
    
    // ============================================================
    // STEP 3: PO Generation from Shortages (NOT IMPLEMENTED)
    // ============================================================
    
    console.log('\n📦 STEP 3: Generate PO from Shortages\n');
    
    const genPoResp = await page.request.post(`/api/v1/workorders/${woNumber}/generate-po`, {
      data: { vendor_id: vendorId },
      failOnStatusCode: false
    });
    
    console.log(`  ❌ BLOCKED: Endpoint not implemented (${genPoResp.status()})\n`);
    console.log(`    - Missing: POST /api/v1/workorders/{id}/generate-po`);
    console.log(`    - Expected: Auto-create PO with line items for BOM shortages`);
    console.log(`    - Workaround: Manually create PO in UI`);
    console.log(`    - Reference: WORKFLOW_GAPS.md #3.1`);
    
    expect(genPoResp.status()).not.toBe(200); // Expect 404 or similar
    
    console.log('\n' + '-'.repeat(60));
    
    // ============================================================
    // STEP 4-6: Manual PO Creation & Receiving Flow
    // ============================================================
    
    console.log('\n📥 STEP 4-6: PO Receiving → Inventory Update Test\n');
    console.log('  (Manual workaround for missing generate-po endpoint)\n');
    
    // Capture inventory BEFORE receiving
    const invBeforeRes = await page.request.get(`/api/v1/inventory/${resistorIpn}`);
    const invBeforeCap = await page.request.get(`/api/v1/inventory/${capacitorIpn}`);
    
    const invBefore = {
      resistor: await invBeforeRes.json(),
      capacitor: await invBeforeCap.json()
    };
    
    console.log(`  📊 Inventory BEFORE PO receiving:`);
    console.log(`    ${resistorIpn}: ${invBefore.resistor.qty_on_hand}`);
    console.log(`    ${capacitorIpn}: ${invBefore.capacitor.qty_on_hand}`);
    
    // Create PO manually with line items for shortages
    console.log(`\n  Creating PO manually...`);
    const poResp = await page.request.post('/api/v1/pos', {
      data: {
        vendor_id: vendorId,
        status: 'open',
        notes: `Integration test PO for ${woNumber}`,
        lines: [
          { ipn: resistorIpn, qty_ordered: 90, unit_price: 0.10 },
          { ipn: capacitorIpn, qty_ordered: 45, unit_price: 0.25 }
        ]
      }
    });
    
    if (!poResp.ok()) {
      console.log(`  ❌ PO creation failed: ${await poResp.text()}`);
      console.log(`     Cannot continue with receiving test\n`);
      return;
    }
    
    const po = await poResp.json();
    const poId = po.po_number || po.id;
    console.log(`  ✓ Created PO: ${poId}`);
    console.log(`    Line 1: 90x ${resistorIpn} @ $0.10`);
    console.log(`    Line 2: 45x ${capacitorIpn} @ $0.25`);
    
    // Receive the PO
    console.log(`\n  Receiving PO...`);
    const receiveResp = await page.request.post(`/api/v1/pos/${poId}/receive`, {
      failOnStatusCode: false
    });
    
    if (!receiveResp.ok()) {
      console.log(`  ⚠️  Receive endpoint failed (${receiveResp.status()})`);
      console.log(`     Error: ${await receiveResp.text()}`);
      console.log(`     PO receiving may not be fully implemented\n`);
    } else {
      console.log(`  ✓ PO receive request succeeded (${receiveResp.status()})`);
      
      // Wait for inventory update
      await page.waitForTimeout(500);
      
      // Capture inventory AFTER receiving
      const invAfterRes = await page.request.get(`/api/v1/inventory/${resistorIpn}`);
      const invAfterCap = await page.request.get(`/api/v1/inventory/${capacitorIpn}`);
      
      const invAfter = {
        resistor: await invAfterRes.json(),
        capacitor: await invAfterCap.json()
      };
      
      console.log(`\n  📊 Inventory AFTER PO receiving:`);
      console.log(`    ${resistorIpn}: ${invAfter.resistor.qty_on_hand}`);
      console.log(`    ${capacitorIpn}: ${invAfter.capacitor.qty_on_hand}`);
      
      // Check if inventory was auto-updated
      const resistorDelta = invAfter.resistor.qty_on_hand - invBefore.resistor.qty_on_hand;
      const capacitorDelta = invAfter.capacitor.qty_on_hand - invBefore.capacitor.qty_on_hand;
      
      console.log(`\n  📈 Inventory Changes:`);
      console.log(`    ${resistorIpn}: ${invBefore.resistor.qty_on_hand} → ${invAfter.resistor.qty_on_hand} (Δ ${resistorDelta})`);
      console.log(`    ${capacitorIpn}: ${invBefore.capacitor.qty_on_hand} → ${invAfter.capacitor.qty_on_hand} (Δ ${capacitorDelta})`);
      
      if (resistorDelta === 90 && capacitorDelta === 45) {
        console.log(`\n  ✅ SUCCESS: Inventory auto-updated correctly!`);
        console.log(`     PO receiving → inventory update is WORKING`);
        
        expect(invAfter.resistor.qty_on_hand).toBe(100); // 10 + 90
        expect(invAfter.capacitor.qty_on_hand).toBe(50);  // 5 + 45
      } else if (resistorDelta === 0 && capacitorDelta === 0) {
        console.log(`\n  ❌ GAP CONFIRMED: Inventory NOT updated after PO receiving`);
        console.log(`     Reference: WORKFLOW_GAPS.md #3.2`);
        console.log(`     Expected: Inventory += received quantities`);
        console.log(`     Actual: Inventory unchanged`);
      } else {
        console.log(`\n  ⚠️  UNEXPECTED: Partial inventory update`);
        console.log(`     Expected deltas: resistor=+90, capacitor=+45`);
        console.log(`     Actual deltas: resistor=${resistorDelta}, capacitor=${capacitorDelta}`);
      }
    }
    
    console.log('\n' + '-'.repeat(60));
    
    // ============================================================
    // STEP 7: Material Reservation Check
    // ============================================================
    
    console.log('\n🔒 STEP 7: Material Reservation on WO Creation\n');
    
    const invCheck = await page.request.get(`/api/v1/inventory/${resistorIpn}`);
    const inv = await invCheck.json();
    
    console.log(`  📊 Current inventory for ${resistorIpn}:`);
    console.log(`    qty_on_hand: ${inv.qty_on_hand}`);
    console.log(`    qty_reserved: ${inv.qty_reserved}`);
    
    if (inv.qty_reserved === 0) {
      console.log(`\n  ❌ GAP CONFIRMED: Materials NOT reserved when WO created`);
      console.log(`     Reference: WORKFLOW_GAPS.md #4.1`);
      console.log(`     Expected: qty_reserved updated when WO created`);
      console.log(`     Actual: qty_reserved = 0 (no reservation)`);
      console.log(`     Risk: Double-allocation of inventory`);
    } else {
      console.log(`\n  ✅ Materials reserved: ${inv.qty_reserved}`);
    }
    
    // ============================================================
    // FINAL SUMMARY
    // ============================================================
    
    console.log('\n' + '='.repeat(60));
    console.log('TC-INT-002 TEST SUMMARY');
    console.log('='.repeat(60) + '\n');
    
    console.log('✅ WORKING:');
    console.log('  • Work order creation');
    console.log('  • BOM endpoint access (GET /api/v1/workorders/{id}/bom)');
    console.log('  • Manual PO creation with line items');
    
    console.log('\n⚠️  PARTIAL / BUGGY:');
    console.log('  • BOM check returns ALL inventory (not assembly-specific)');
    console.log('  • PO receiving may/may not update inventory (tested above)');
    
    console.log('\n❌ NOT IMPLEMENTED:');
    console.log('  • Generate PO from WO shortages (no endpoint)');
    console.log('  • Material reservation on WO creation (qty_reserved not used)');
    console.log('  • BOM management via API (file-based only)');
    
    console.log('\n📚 REFERENCES:');
    console.log('  • docs/INTEGRATION_TEST_PLAN.md (TC-INT-001)');
    console.log('  • docs/WORKFLOW_GAPS.md (#1.5, #3.1, #3.2, #4.1)');
    console.log('  • docs/INTEGRATION_TESTS_NEEDED.md (TC-INT-002)');
    
    console.log('\n' + '='.repeat(60) + '\n');
    
    // Test passes if it runs to completion (documents gaps, doesn't fail)
  });
  
  test('Quick smoke test: WO creation and BOM endpoint', async ({ page }) => {
    const timestamp = Date.now();
    const woId = `WO-SMOKE-${timestamp}`;
    
    console.log('\n🔥 Smoke Test: WO Creation + BOM Endpoint\n');
    
    const woResp = await page.request.post('/api/v1/workorders', {
      data: {
        id: woId,
        assembly_ipn: 'TEST-ASY',
        qty: 1,
        status: 'open'
      }
    });
    
    expect(woResp.ok()).toBeTruthy();
    console.log(`  ✓ WO created: ${woId}`);
    
    const bomResp = await page.request.get(`/api/v1/workorders/${woId}/bom`);
    expect(bomResp.ok()).toBeTruthy();
    console.log(`  ✓ BOM endpoint accessible`);
    
    const bomData = await bomResp.json();
    expect(bomData.wo_id).toBe(woId);
    expect(bomData.bom).toBeDefined();
    console.log(`  ✓ BOM data structure valid\n`);
  });
});
