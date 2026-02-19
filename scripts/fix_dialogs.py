#!/usr/bin/env python3
"""
Fix missing DialogDescription in React components for accessibility.
Adds DialogDescription import and placeholder descriptions to dialogs.
"""

import re
import os
from pathlib import Path

FRONTEND_SRC = Path(__file__).parent.parent / "frontend" / "src"

# Dialog descriptions by context
DIALOG_DESCRIPTIONS = {
    "Create": "Fill out the form below to create a new {entity}.",
    "Edit": "Update the information for this {entity}.",
    "Delete": "This action cannot be undone. Are you sure you want to delete this {entity}?",
    "Upload": "Select and upload files for this {entity}.",
    "Import": "Import data from a CSV file.",
    "Export": "Export data to a file.",
    "Receive": "Record received items and update inventory.",
    "Generate": "Generate a new {entity} based on the selected data.",
    "Quick Receive": "Quickly receive inventory for a specific part.",
    "Bulk Edit": "Edit multiple items at once.",
    "Inspect": "Inspect and record quality information for this {entity}.",
}

def add_dialog_description_import(content: str) -> tuple[str, bool]:
    """Add DialogDescription to Dialog imports if missing."""
    # Check if already imported
    if "DialogDescription" in content:
        return content, False
    
    # Find Dialog import statement
    dialog_import_pattern = r'(import\s+\{[^}]*DialogContent,)([^}]*\}\s+from\s+["\']\.\.\/components\/ui\/dialog["\'])'
    match = re.search(dialog_import_pattern, content, re.MULTILINE)
    
    if match:
        # Insert DialogDescription after DialogContent
        new_import = match.group(1) + '\n  DialogDescription,' + match.group(2)
        content = content[:match.start()] + new_import + content[match.end():]
        return content, True
    
    return content, False

def add_dialog_descriptions(content: str, filename: str) -> tuple[str, int]:
    """Add DialogDescription to DialogHeader sections that are missing it."""
    count = 0
    
    # Pattern: DialogHeader with DialogTitle but no DialogDescription
    pattern = r'(<DialogHeader>\s*<DialogTitle>([^<]+)</DialogTitle>\s*)(</DialogHeader>)'
    
    def replace_func(match):
        nonlocal count
        title_text = match.group(2).strip()
        
        # Determine appropriate description based on title
        desc = "Complete the form below."  # Default
        
        for keyword, template in DIALOG_DESCRIPTIONS.items():
            if keyword.lower() in title_text.lower():
                # Try to extract entity name
                entity = re.sub(r'(Create|Edit|Delete|Upload|Import|Export|Generate|Receive|Inspect)\s+', '', title_text, flags=re.IGNORECASE).strip()
                if not entity:
                    entity = "item"
                desc = template.format(entity=entity.lower())
                break
        
        count += 1
        return (match.group(1) + 
                f'\n              <DialogDescription>\n                {desc}\n              </DialogDescription>\n              ' +
                match.group(3))
    
    new_content = re.sub(pattern, replace_func, content)
    return new_content, count

def process_file(filepath: Path) -> dict:
    """Process a single file."""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        
        # Add import
        content, import_added = add_dialog_description_import(content)
        
        # Add descriptions
        content, desc_count = add_dialog_descriptions(content, filepath.name)
        
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            
            return {
                'file': filepath.name,
                'import_added': import_added,
                'descriptions_added': desc_count,
                'success': True
            }
        
        return {'file': filepath.name, 'success': False, 'reason': 'No changes needed'}
    
    except Exception as e:
        return {'file': filepath.name, 'success': False, 'error': str(e)}

def main():
    """Main function to process all TypeScript/TSX files."""
    pages_dir = FRONTEND_SRC / "pages"
    components_dir = FRONTEND_SRC / "components"
    
    results = []
    
    # Process page files
    for filepath in pages_dir.glob("*.tsx"):
        if "test" not in filepath.name.lower():
            result = process_file(filepath)
            if result.get('success') or result.get('error'):
                results.append(result)
    
    # Process component files
    for filepath in components_dir.glob("*.tsx"):
        if "test" not in filepath.name.lower():
            result = process_file(filepath)
            if result.get('success') or result.get('error'):
                results.append(result)
    
    # Print results
    print("\n=== Dialog Description Fix Results ===\n")
    
    total_imports = sum(1 for r in results if r.get('import_added'))
    total_descriptions = sum(r.get('descriptions_added', 0) for r in results)
    total_files = len(results)
    
    print(f"Files processed: {total_files}")
    print(f"Imports added: {total_imports}")
    print(f"Descriptions added: {total_descriptions}")
    print()
    
    if results:
        print("Details:")
        for r in results:
            if r.get('success'):
                print(f"  ✓ {r['file']}: +{r.get('descriptions_added', 0)} descriptions" + 
                      (" [import added]" if r.get('import_added') else ""))
            elif r.get('error'):
                print(f"  ✗ {r['file']}: {r['error']}")

if __name__ == "__main__":
    main()
