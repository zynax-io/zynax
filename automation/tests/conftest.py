"""
conftest.py — stub fixtures for automation/tests/.

The zynax_client fixture is intentionally unimplemented: it raises pytest.fail
so that tests depending on a live platform xfail (not error) when the platform
is not running. Because all tests that use this fixture are wrapped in
pytest.mark.xfail(strict=True), the failure is treated as XFAIL.
"""

import pytest


@pytest.fixture
def zynax_client():
    """Stub for a live Zynax platform client.

    Raises pytest.fail so that xfail-wrapped tests are marked XFAIL rather
    than ERROR. Replace with a real client once the platform is deployed.
    """
    pytest.fail("zynax platform not running — Wave 4 prerequisite not met")
