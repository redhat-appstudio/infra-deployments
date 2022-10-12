
ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"/..
# compare template with preview.env
# if any new or different exports
# print a warning
# If any new ENV vars are added to the template
# users can tell they need to update their preview.env
TEMPLATEENV=$(mktemp)
YOURENV=$(mktemp)
cat $ROOT/hack/preview-template.env | \
    cut -d '=' -f 1 | \
    grep -v "#" | grep "\S" | sort > $TEMPLATEENV
cat $ROOT/hack/preview.env | \
    cut -d '=' -f 1 | \
    grep -v "#" | grep "\S" | sort > $YOURENV 
echo "Comparing preview.env and preview-template.env"
if diff $TEMPLATEENV $YOURENV; then
    echo "preview.env and preview-template.env are consistent"
else
    echo "Warning preview.env has different exports than  preview-template.env"
    echo "There may be new features you need to enable in preview mode."
fi