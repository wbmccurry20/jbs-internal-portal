-- Update IT-Will user to have owner role for reconciliation access
UPDATE users 
SET role = 'owner' 
WHERE email = 'hello@itwill.dev';

-- Verify the update
SELECT email, role FROM users WHERE email = 'hello@itwill.dev';
