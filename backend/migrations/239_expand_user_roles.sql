-- Expand the user role vocabulary while preserving all existing administrator access.
-- Historical admin rows are the full administrators and therefore become super_admin.
UPDATE users
SET role = 'super_admin'
WHERE role = 'admin';

COMMENT ON COLUMN users.role IS
  'Authorization role: super_admin, admin, user, or enterprise_user';
