#!/bin/bash
# Fix missing DialogDescription in all dialog components for accessibility

cd "$(dirname "$0")/../frontend/src"

# Files that need DialogDescription import added
files_needing_import=(
  "pages/WorkOrders.tsx"
  "pages/Inventory.tsx"
  "pages/Devices.tsx"
  "pages/ECOs.tsx"
  "pages/Users.tsx"
  "pages/PODetail.tsx"
  "pages/WorkOrderDetail.tsx"
  "pages/Firmware.tsx"
  "pages/Testing.tsx"
  "pages/InventoryDetail.tsx"
  "pages/Vendors.tsx"
  "pages/Quotes.tsx"
  "pages/NCRs.tsx"
  "pages/APIKeys.tsx"
  "pages/RMAs.tsx"
  "pages/Documents.tsx"
  "components/BulkEditDialog.tsx"
)

echo "Adding DialogDescription imports..."
for file in "${files_needing_import[@]}"; do
  if [ -f "$file" ]; then
    # Check if DialogDescription is already imported
    if ! grep -q "DialogDescription" "$file"; then
      # Add DialogDescription to the import statement
      if grep -q "Dialog," "$file"; then
        sed -i.bak 's/DialogContent,/DialogContent,\n  DialogDescription,/' "$file"
        echo "✓ Added DialogDescription import to $file"
      fi
    fi
  fi
done

echo ""
echo "Manual fix required: Add <DialogDescription> after <DialogTitle> in each dialog."
echo "Pattern:"
echo "  <DialogHeader>"
echo "    <DialogTitle>Your Title</DialogTitle>"
echo "    <DialogDescription>Your description for screen readers</DialogDescription>"
echo "  </DialogHeader>"
echo ""
echo "Files to manually update: ${#files_needing_import[@]}"
